package notify

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"catalogsvc/internal/model"
)

// Subscriber receives change events after catalog mutations.
type Subscriber interface {
	Deliver(ctx context.Context, event model.ChangeEvent) error
}

// ErrDeliveryFailed is returned when a subscriber cannot be reached.
var ErrDeliveryFailed = errors.New("notify: delivery failed")

// defaultMaxRetries is the number of delivery attempts made for a single
// subscriber before a failure is considered permanent. The first attempt plus
// this many retries yield at most defaultMaxRetries+1 total tries.
const defaultMaxRetries = 3

// defaultRetryBackoff is the wait between retry attempts for a slow or
// temporarily unreachable subscriber. It is short because subscribers are
// in-process; callers that need jitter or larger backoffs can override Sleep.
const defaultRetryBackoff = 50 * time.Millisecond

// failure tracks the retry state for one (event, subscriber) pair.
type failure struct {
	attempts int
	err      error
}

// Notifier fans catalog events out to subscribers with retry bookkeeping.
type Notifier struct {
	mu          sync.RWMutex
	subscribers map[string]Subscriber
	attempts    map[string]int  // max attempts across subscribers, by event key
	lastErr     map[string]error // worst error across subscribers, by event key
	failures    map[string]failure // by subscriber id, for the last pushed event

	// MaxRetries overrides the default retry count when non-zero.
	MaxRetries int
	// Sleep is invoked between retry attempts. Defaults to a fixed backoff;
	// tests may replace it to avoid real waiting.
	Sleep func(d time.Duration)
}

// New creates an empty notifier.
func New() *Notifier {
	return &Notifier{
		subscribers: make(map[string]Subscriber),
		attempts:    make(map[string]int),
		lastErr:     make(map[string]error),
		failures:    make(map[string]failure),
	}
}

// Subscribe registers a subscriber under a stable id.
func (n *Notifier) Subscribe(id string, sub Subscriber) {
	n.mu.Lock()
	n.subscribers[id] = sub
	n.mu.Unlock()
}

// Unsubscribe removes a subscriber by id.
func (n *Notifier) Unsubscribe(id string) {
	n.mu.Lock()
	delete(n.subscribers, id)
	n.mu.Unlock()
}

// SubscriberCount returns the number of registered subscribers.
func (n *Notifier) SubscriberCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.subscribers)
}

// Push delivers an event to every subscriber, retrying transient failures up
// to MaxRetries times. Delivery errors are recorded in the bookkeeping maps
// and, when a subscriber remains unreachable after all retries, surfaced to
// the caller as a wrapped ErrDeliveryFailed so downstream consumers can
// report and react instead of silently dropping the event.
func (n *Notifier) Push(ctx context.Context, event model.ChangeEvent) error {
	n.mu.RLock()
	ids := make([]string, 0, len(n.subscribers))
	subs := make([]Subscriber, 0, len(n.subscribers))
	for id, sub := range n.subscribers {
		ids = append(ids, id)
		subs = append(subs, sub)
	}
	n.mu.RUnlock()

	maxRetries := n.maxRetries()
	totalAttempts := maxRetries + 1 // first attempt plus MaxRetries retries
	sleep := n.sleep()

	var failed []string
	var firstErr error

	for i, sub := range subs {
		id := ids[i]
		var err error
		tries := 0
		for tries = 1; tries <= totalAttempts; tries++ {
			err = sub.Deliver(ctx, event)
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				// The context is cancelled: no point retrying further.
				break
			}
			if tries < totalAttempts {
				sleep(n.retryBackoff(tries))
			}
		}
		// On a normal exit the for-loop has already incremented tries past
		// the final iteration; clamp to the actual attempt count so the
		// bookkeeping reflects what really happened.
		if tries > totalAttempts {
			tries = totalAttempts
		}
		n.record(id, event.Key(), tries, err)
		if err != nil {
			failed = append(failed, id)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if len(failed) > 0 {
		return fmtDeliveryError(failed, firstErr)
	}
	return nil
}

// record stores retry bookkeeping under the notifier lock. Per-subscriber
// state is keyed by subscriber id; the legacy event-keyed maps expose the
// worst-case view across subscribers for backward compatibility.
func (n *Notifier) record(subID, eventKey string, attempts int, err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.failures[subID] = failure{attempts: attempts, err: err}

	// Maintain the event-keyed aggregates consumed by Attempts/LastError.
	prev := n.attempts[eventKey]
	if attempts > prev {
		n.attempts[eventKey] = attempts
	}
	if err != nil {
		n.lastErr[eventKey] = err
	} else if _, hadErr := n.lastErr[eventKey]; hadErr {
		// A successful delivery for this event clears the stale aggregate only
		// when no other subscriber has a recorded failure for the same event.
		if !n.anyFailureForEventLocked(eventKey) {
			delete(n.lastErr, eventKey)
		}
	}
}

// anyFailureForEventLocked reports whether any per-subscriber failure still
// holds an error. Caller must hold n.mu.
func (n *Notifier) anyFailureForEventLocked(eventKey string) bool {
	for _, f := range n.failures {
		if f.err != nil {
			return true
		}
	}
	return false
}

// Attempts returns how many delivery attempts were made for an event, taken
// as the maximum across subscribers. It is retained for backward-compatible
// observability of the last pushed event.
func (n *Notifier) Attempts(eventKey string) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.attempts[eventKey]
}

// LastError returns the last recorded delivery error for an event, aggregated
// across subscribers. It is retained for backward-compatible observability of
// the last pushed event.
func (n *Notifier) LastError(eventKey string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastErr[eventKey]
}

// PendingFailures lists event keys with a recorded delivery failure. It is
// retained for backward compatibility; prefer PendingSubscriberFailures for
// the precise per-subscriber view used to drive replay.
func (n *Notifier) PendingFailures() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	keys := make([]string, 0, len(n.lastErr))
	for key, err := range n.lastErr {
		if err != nil {
			keys = append(keys, key)
		}
	}
	return keys
}

// PendingSubscriberFailures lists the subscriber ids that still hold a
// recorded delivery failure for the last pushed event. Callers use this to
// drive targeted replay once the failing subscriber recovers.
func (n *Notifier) PendingSubscriberFailures() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	ids := make([]string, 0, len(n.failures))
	for id, f := range n.failures {
		if f.err != nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// maxRetries returns the configured retry count, applying the default when
// none was set.
func (n *Notifier) maxRetries() int {
	if n.MaxRetries > 0 {
		return n.MaxRetries
	}
	return defaultMaxRetries
}

// sleep returns the inter-retry sleep function, defaulting to time.Sleep.
func (n *Notifier) sleep() func(time.Duration) {
	if n.Sleep != nil {
		return n.Sleep
	}
	return time.Sleep
}

// retryBackoff returns the wait before the next attempt. It grows linearly so
// a briefly unavailable subscriber gets a chance to recover without delaying
// the fast path.
func (n *Notifier) retryBackoff(tries int) time.Duration {
	return defaultRetryBackoff * time.Duration(tries)
}

// fmtDeliveryError wraps the first delivery error with ErrDeliveryFailed and
// names the subscribers that failed, so callers can distinguish a delivery
// failure from other errors via errors.Is(err, ErrDeliveryFailed).
func fmtDeliveryError(failed []string, first error) error {
	return deliveryError{failed: failed, cause: first}
}

// deliveryError is an error returned when one or more subscribers could not
// be reached after all retries.
type deliveryError struct {
	failed []string
	cause  error
}

func (e deliveryError) Error() string {
	if e.cause == nil {
		return "notify: delivery failed"
	}
	var b strings.Builder
	b.WriteString("notify: delivery failed for subscriber(s) ")
	b.WriteString(strings.Join(e.failed, ", "))
	b.WriteString(": ")
	b.WriteString(e.cause.Error())
	return b.String()
}

func (e deliveryError) Unwrap() error {
	if e.cause != nil {
		return e.cause
	}
	return ErrDeliveryFailed
}

func (e deliveryError) Is(target error) bool {
	if target == ErrDeliveryFailed {
		return true
	}
	// Allow errors.Is to match the underlying cause as well.
	return false
}

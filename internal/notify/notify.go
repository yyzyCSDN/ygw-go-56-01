package notify

import (
	"context"
	"errors"
	"sync"

	"catalogsvc/internal/model"
)

// Subscriber receives change events after catalog mutations.
type Subscriber interface {
	Deliver(ctx context.Context, event model.ChangeEvent) error
}

// ErrDeliveryFailed is returned when a subscriber cannot be reached.
var ErrDeliveryFailed = errors.New("notify: delivery failed")

// Notifier fans catalog events out to subscribers with retry bookkeeping.
type Notifier struct {
	mu          sync.RWMutex
	subscribers map[string]Subscriber
	attempts    map[string]int
	lastErr     map[string]error
}

// New creates an empty notifier.
func New() *Notifier {
	return &Notifier{
		subscribers: make(map[string]Subscriber),
		attempts:    make(map[string]int),
		lastErr:     make(map[string]error),
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

// Push delivers an event to every subscriber once. Delivery errors are logged
// in the bookkeeping maps and otherwise swallowed by the caller.
func (n *Notifier) Push(ctx context.Context, event model.ChangeEvent) error {
	n.mu.RLock()
	subs := make([]Subscriber, 0, len(n.subscribers))
	for _, sub := range n.subscribers {
		subs = append(subs, sub)
	}
	n.mu.RUnlock()

	for _, sub := range subs {
		err := sub.Deliver(ctx, event)
		n.record(event.Key(), 1, err)
	}
	return nil
}

// record stores retry bookkeeping under the notifier lock.
func (n *Notifier) record(eventKey string, attempts int, err error) {
	n.mu.Lock()
	n.attempts[eventKey] = attempts
	if err != nil {
		n.lastErr[eventKey] = err
	}
	n.mu.Unlock()
}

// Attempts returns how many delivery attempts were made for an event.
func (n *Notifier) Attempts(eventKey string) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.attempts[eventKey]
}

// LastError returns the last recorded delivery error for an event.
func (n *Notifier) LastError(eventKey string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastErr[eventKey]
}

// PendingFailures lists event keys with a recorded delivery failure.
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

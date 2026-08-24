package notify

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"catalogsvc/internal/model"
)

// failingSub fails for the first N Deliver calls then succeeds.
type failingSub struct {
	name     string
	failN    int32
	calls    int32
	finalErr error
}

func (f *failingSub) Deliver(_ context.Context, _ model.ChangeEvent) error {
	n := atomic.AddInt32(&f.calls, 1)
	if n <= f.failN {
		return f.finalErr
	}
	return nil
}

func (f *failingSub) Calls() int { return int(atomic.LoadInt32(&f.calls)) }

// newRetryNotifier returns a notifier whose retry sleep is a no-op so tests
// do not block on real backoff.
func newRetryNotifier() *Notifier {
	n := New()
	n.Sleep = func(time.Duration) {}
	return n
}

func TestPushRetriesTransientFailure(t *testing.T) {
	n := newRetryNotifier()
	n.MaxRetries = 3
	// Fail three times then succeed: the fourth attempt should succeed.
	// (one initial attempt plus MaxRetries retries = 4 total.)
	sub := &failingSub{name: "flaky", failN: 3, finalErr: errors.New("temp")}
	n.Subscribe("flaky", sub)

	event := model.ChangeEvent{TableID: "t1", Version: 1, Type: model.EventSchemaChanged}
	if err := n.Push(context.Background(), event); err != nil {
		t.Fatalf("transient failure should recover: %v", err)
	}
	if got := sub.Calls(); got != 4 {
		t.Fatalf("expected 4 attempts, got %d", got)
	}
	if n.Attempts(event.Key()) != 4 {
		t.Fatalf("attempts bookkeeping wrong: %d", n.Attempts(event.Key()))
	}
	if err := n.LastError(event.Key()); err != nil {
		t.Fatalf("last error should be cleared after success: %v", err)
	}
	if pending := n.PendingSubscriberFailures(); len(pending) != 0 {
		t.Fatalf("no pending failures expected, got %v", pending)
	}
}

func TestPushReportsPermanentFailure(t *testing.T) {
	n := newRetryNotifier()
	n.MaxRetries = 3
	boom := errors.New("boom")
	sub := &failingSub{name: "dead", failN: 99, finalErr: boom}
	n.Subscribe("dead", sub)

	event := model.ChangeEvent{TableID: "t2", Version: 1, Type: model.EventSchemaChanged}
	err := n.Push(context.Background(), event)
	if err == nil {
		t.Fatalf("permanent failure must be reported")
	}
	if !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("error must wrap ErrDeliveryFailed, got: %v", err)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("error must expose underlying cause, got: %v", err)
	}
	// One initial attempt plus MaxRetries retries = 4 total.
	if got := sub.Calls(); got != 4 {
		t.Fatalf("expected 4 attempts before giving up, got %d", got)
	}
	if n.Attempts(event.Key()) != 4 {
		t.Fatalf("attempts bookkeeping wrong: %d", n.Attempts(event.Key()))
	}
	if n.LastError(event.Key()) == nil {
		t.Fatalf("last error must be recorded for permanent failure")
	}
	pending := n.PendingSubscriberFailures()
	if len(pending) != 1 || pending[0] != "dead" {
		t.Fatalf("pending failures wrong: %v", pending)
	}
}

func TestPushMixedSubscribersIsolateFailures(t *testing.T) {
	n := newRetryNotifier()
	n.MaxRetries = 2
	// "ok" always succeeds; "bad" always fails.
	ok := &failingSub{name: "ok", failN: 0, finalErr: nil}
	bad := &failingSub{name: "bad", failN: 99, finalErr: errors.New("down")}
	n.Subscribe("ok", ok)
	n.Subscribe("bad", bad)

	event := model.ChangeEvent{TableID: "t3", Version: 1, Type: model.EventTableCreated}
	err := n.Push(context.Background(), event)
	if err == nil {
		t.Fatalf("partial failure must be reported")
	}
	if !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("must wrap ErrDeliveryFailed: %v", err)
	}
	// The healthy subscriber must still receive the event exactly once.
	if got := ok.Calls(); got != 1 {
		t.Fatalf("healthy subscriber should be called once, got %d", got)
	}
	// The failing subscriber must exhaust its retries (1 + MaxRetries = 3).
	if got := bad.Calls(); got != 3 {
		t.Fatalf("failing subscriber should be retried, got %d", got)
	}
	// Pending failures list the failing subscriber only.
	pending := n.PendingSubscriberFailures()
	if len(pending) != 1 || pending[0] != "bad" {
		t.Fatalf("pending should list bad only: %v", pending)
	}
}

func TestPushCancelledContextDoesNotRetry(t *testing.T) {
	n := newRetryNotifier()
	n.MaxRetries = 5
	sub := &failingSub{name: "dead", failN: 99, finalErr: errors.New("down")}
	n.Subscribe("dead", sub)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the first deliver so ctx.Err() is non-nil

	event := model.ChangeEvent{TableID: "t4", Version: 1, Type: model.EventVersionPublish}
	_ = n.Push(ctx, event)
	// The first attempt is made, but the cancelled context stops further retries.
	if got := sub.Calls(); got > 1 {
		t.Fatalf("cancelled context must not trigger retries, got %d calls", got)
	}
}

func TestPushEmptySubscribersIsSuccess(t *testing.T) {
	n := newRetryNotifier()
	n.MaxRetries = 3
	event := model.ChangeEvent{TableID: "t5", Version: 1, Type: model.EventTableCreated}
	if err := n.Push(context.Background(), event); err != nil {
		t.Fatalf("push with no subscribers must succeed: %v", err)
	}
	if n.Attempts(event.Key()) != 0 {
		t.Fatalf("attempts should be 0 with no subscribers, got %d", n.Attempts(event.Key()))
	}
}

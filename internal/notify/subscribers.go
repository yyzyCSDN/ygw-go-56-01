package notify

import (
	"context"
	"sync"

	"catalogsvc/internal/model"
)

// FuncSubscriber adapts a function to the Subscriber interface.
type FuncSubscriber func(ctx context.Context, event model.ChangeEvent) error

// Deliver implements Subscriber.
func (f FuncSubscriber) Deliver(ctx context.Context, event model.ChangeEvent) error {
	return f(ctx, event)
}

// LogSubscriber records delivered events for audit and replay.
type LogSubscriber struct {
	mu     sync.Mutex
	events []model.ChangeEvent
}

// NewLogSubscriber creates an empty event log.
func NewLogSubscriber() *LogSubscriber {
	return &LogSubscriber{}
}

// Deliver appends the event to the in-memory log.
func (l *LogSubscriber) Deliver(_ context.Context, event model.ChangeEvent) error {
	l.mu.Lock()
	l.events = append(l.events, event)
	l.mu.Unlock()
	return nil
}

// Events returns a copy of all delivered events in order.
func (l *LogSubscriber) Events() []model.ChangeEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	next := make([]model.ChangeEvent, len(l.events))
	copy(next, l.events)
	return next
}

// Count reports how many events have been delivered.
func (l *LogSubscriber) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.events)
}

// Replay delivers a previously captured event to another subscriber.
func Replay(ctx context.Context, target Subscriber, event model.ChangeEvent) error {
	return target.Deliver(ctx, event)
}

package notify

import (
	"context"
	"errors"
	"testing"

	"catalogsvc/internal/model"
)

func TestNotifyFailureNotSwallowed(t *testing.T) {
	notifier := New()
	calls := 0
	notifier.Subscribe("downstream", FuncSubscriber(func(_ context.Context, _ model.ChangeEvent) error {
		calls++
		return errors.New("downstream endpoint unavailable")
	}))
	event := model.ChangeEvent{TableID: "t1", Version: 4, Type: model.EventSchemaChanged}
	err := notifier.Push(context.Background(), event)
	if err == nil {
		t.Fatalf("delivery failure was swallowed")
	}
	if calls < 2 {
		t.Fatalf("failed delivery was not retried: %d calls", calls)
	}
	if attempts := notifier.Attempts(event.Key()); attempts < 2 {
		t.Fatalf("retry not recorded: %d attempts", attempts)
	}
	if notifier.LastError(event.Key()) == nil {
		t.Fatalf("delivery error not recorded")
	}
	if pending := notifier.PendingFailures(); len(pending) != 1 {
		t.Fatalf("pending failures wrong: %v", pending)
	}
}

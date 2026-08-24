package notify

import (
	"context"
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
)

func TestSuccessfulPush(t *testing.T) {
	notifier := New()
	logger := NewLogSubscriber()
	notifier.Subscribe("logger", logger)
	if notifier.SubscriberCount() != 1 {
		t.Fatalf("subscriber count wrong")
	}
	event := model.ChangeEvent{TableID: "t1", Version: 3, Type: model.EventSchemaChanged}
	if err := notifier.Push(context.Background(), event); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if logger.Count() != 1 || logger.Events()[0].TableID != "t1" {
		t.Fatalf("event not delivered: %+v", logger.Events())
	}
	if notifier.Attempts(event.Key()) != 1 {
		t.Fatalf("attempts wrong: %d", notifier.Attempts(event.Key()))
	}
	notifier.Unsubscribe("logger")
	if notifier.SubscriberCount() != 0 {
		t.Fatalf("unsubscribe failed")
	}
}

func TestFuncSubscriberAndReplay(t *testing.T) {
	notifier := New()
	var delivered int
	notifier.Subscribe("fn", FuncSubscriber(func(_ context.Context, _ model.ChangeEvent) error {
		delivered++
		return nil
	}))
	event := model.ChangeEvent{TableID: "t2", Version: 1, Type: model.EventTableCreated}
	if err := notifier.Push(context.Background(), event); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if delivered != 1 {
		t.Fatalf("func subscriber not called")
	}
	logger := NewLogSubscriber()
	if err := Replay(context.Background(), logger, event); err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if logger.Count() != 1 {
		t.Fatalf("replay did not deliver")
	}
}

func TestSnapshotBuilder(t *testing.T) {
	cat := catalog.New()
	if err := cat.Register(&model.Table{ID: "t1", Name: "orders", Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	builder := NewSnapshotBuilder(cat)
	snapshot, err := builder.Build("t1", 1)
	if err != nil || snapshot.Table.ID != "t1" || snapshot.Version != 1 {
		t.Fatalf("build failed: %v %+v", err, snapshot)
	}
	all := builder.BuildAll()
	if len(all) != 1 || all[0].Table.Name != "orders" {
		t.Fatalf("build all failed: %+v", all)
	}
}

package database

import (
	"context"
	"testing"
)

func TestAfterCommitQueuesInsideTransactionAndDoesNotExecuteOnRollback(t *testing.T) {
	ctx := context.WithValue(context.Background(), afterCommitKey{}, &commitHooks{})
	called := 0
	AfterCommit(ctx, func(context.Context) { called++ })
	if called != 0 {
		t.Fatal("event executed before transaction completion")
	}
	// A rollback doesn't drain hooks: queued work must not be published.
	if called != 0 {
		t.Fatal("rolled back transaction dispatched event")
	}
}

func TestAfterCommitDrainsInOrderAndMasksFinishedTransaction(t *testing.T) {
	hooks := &commitHooks{}
	ctx := context.WithValue(context.Background(), afterCommitKey{}, hooks)
	order := []int{}
	AfterCommit(ctx, func(context.Context) { order = append(order, 1) })
	AfterCommit(ctx, func(context.Context) { order = append(order, 2) })
	for _, fn := range hooks.drain() {
		fn(context.Background())
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("wrong callback ordering: %v", order)
	}
	if len(hooks.drain()) != 0 {
		t.Fatal("callbacks must be drained exactly once")
	}
}

package event

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type recordCreated struct{ ID string }

func TestBusDeliversTypedPayloadToMultipleSubscribers(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Name: "core.record.created", Module: "core"}); err != nil {
		t.Fatal(err)
	}
	bus := NewBus(registry)
	seen := []string{}
	if err := Subscribe(bus, "core.record.created", "first", func(_ context.Context, payload recordCreated) error {
		seen = append(seen, "first:"+payload.ID)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := Subscribe(bus, "core.record.created", "second", func(_ context.Context, payload recordCreated) error {
		seen = append(seen, "second:"+payload.ID)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := Publish(bus, context.Background(), "core.record.created", recordCreated{ID: "r1"}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || seen[0] != "first:r1" || seen[1] != "second:r1" {
		t.Fatalf("unexpected subscribers: %v", seen)
	}
	if err := Publish(bus, context.Background(), "core.record.created", "wrong"); err == nil {
		t.Fatal("expected typed payload error")
	}
	if err := Subscribe(bus, "core.record.created", "first", func(context.Context, recordCreated) error { return nil }); err == nil {
		t.Fatal("expected duplicate subscriber error")
	}
	if err := Subscribe(bus, "core.record.created", "failing", func(context.Context, recordCreated) error { return fmt.Errorf("subscriber failed") }); err != nil {
		t.Fatal(err)
	}
	if err := Publish(bus, context.Background(), "core.record.created", recordCreated{ID: "r2"}); err == nil || !strings.Contains(err.Error(), "subscriber failed") {
		t.Fatalf("unexpected subscriber error: %v", err)
	}
}

package application

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
)

type logCapture struct{ messages []string }

func (h *logCapture) Enabled(context.Context, slog.Level) bool { return true }
func (h *logCapture) Handle(_ context.Context, record slog.Record) error {
	h.messages = append(h.messages, record.Message)
	return nil
}
func (h *logCapture) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *logCapture) WithGroup(string) slog.Handler      { return h }

func TestPostCommitCRMEventFailureIsLogged(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{Name: "crm.opportunity.created", Module: "crm"}); err != nil {
		t.Fatal(err)
	}
	bus := event.NewBus(registry)
	if err := event.Subscribe(bus, "crm.opportunity.created", "failing-fixture", func(context.Context, domain.Event) error {
		return errors.New("fixture subscriber failure")
	}); err != nil {
		t.Fatal(err)
	}
	capture := &logCapture{}
	service := NewWithDependencies(Dependencies{Events: bus, Logger: slog.New(capture)})
	service.publish(context.Background(), "crm.opportunity.created", uuid.New(), uuid.New())
	if len(capture.messages) != 1 || capture.messages[0] != "post-commit event publication failed" {
		t.Fatalf("expected failed post-commit event to be visible in logs, got %+v", capture.messages)
	}
}

package domain

import (
	"testing"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
)

func TestRuntimeFieldValidation(t *testing.T) {
	valid := RuntimeField{Key: "customer_tier", Label: "Customer tier", Type: field.Enum, Options: []string{"standard", "enterprise"}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid field: %v", err)
	}
	if err := (RuntimeField{Key: "bad-key", Label: "Bad", Type: field.String}).Validate(); err == nil {
		t.Fatal("expected invalid key to fail")
	}
	if err := (RuntimeField{Key: "tier", Label: "Tier", Type: field.Enum}).Validate(); err == nil {
		t.Fatal("expected selection without options to fail")
	}
}

func TestSavedViewRejectsUnsafeOperators(t *testing.T) {
	fields := map[string]struct{}{"name": {}}
	view := SavedView{Name: "Bad view", Columns: []string{"name"}, Filters: []Filter{{Field: "name", Operator: "sql"}}}
	if err := view.Validate(fields); err == nil {
		t.Fatal("expected unsupported filter operator to fail")
	}
}

func TestValidValue(t *testing.T) {
	if !ValidValue(field.Date, nil, "2026-10-08") {
		t.Fatal("expected ISO date to be valid")
	}
	if ValidValue(field.Date, nil, "08/10/2026") {
		t.Fatal("expected non-ISO date to fail")
	}
	if !ValidValue(field.Enum, []string{"new", "qualified"}, "qualified") {
		t.Fatal("expected allowed enum value to be valid")
	}
}

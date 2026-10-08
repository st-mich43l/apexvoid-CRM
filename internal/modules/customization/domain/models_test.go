package domain

import (
	"math"
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
	fields := map[string]EffectiveField{"name": {Key: "name", Type: field.String}}
	view := SavedView{Name: "Bad view", Columns: []string{"name"}, Filters: []Filter{{Field: "name", Operator: "sql"}}}
	if err := view.Validate(fields); err == nil {
		t.Fatal("expected unsupported filter operator to fail")
	}
}

func TestSavedViewValidatesFilterValueTypes(t *testing.T) {
	fields := map[string]EffectiveField{
		"name":      {Key: "name", Type: field.String},
		"employee":  {Key: "employee", Type: field.Integer},
		"segment":   {Key: "segment", Type: field.Enum, Options: []string{"mid", "enterprise"}},
		"follow_up": {Key: "follow_up", Type: field.Date},
	}
	valid := SavedView{Name: "Typed view", Filters: []Filter{{Field: "employee", Operator: "eq", Value: float64(12)}, {Field: "segment", Operator: "in", Value: []any{"mid"}}}}
	if err := valid.Validate(fields); err != nil {
		t.Fatalf("expected typed filters to validate: %v", err)
	}
	invalid := SavedView{Name: "Invalid typed view", Filters: []Filter{{Field: "employee", Operator: "eq", Value: "12"}}}
	if err := invalid.Validate(fields); err == nil {
		t.Fatal("expected integer filter with string value to fail")
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

func TestIntegerRequiresWholeFiniteSafeNumber(t *testing.T) {
	for _, value := range []any{1.25, -0.1, 9007199254740992.0, 1e100} {
		if ValidValue(field.Integer, nil, value) {
			t.Errorf("integer field accepted invalid value %v", value)
		}
	}
	for _, value := range []any{0.0, -3.0, 28.0, 9007199254740991.0} {
		if !ValidValue(field.Integer, nil, value) {
			t.Errorf("integer field rejected valid value %v", value)
		}
	}
	if ValidValue(field.Decimal, nil, math.Inf(1)) || ValidValue(field.Decimal, nil, math.NaN()) {
		t.Fatal("decimal must reject non-finite values")
	}
}

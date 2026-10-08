package application

import (
	"testing"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
)

func TestContactLegacyFieldTypeMapping(t *testing.T) {
	cases := map[string]field.Type{
		"text":      field.String,
		"number":    field.Decimal,
		"boolean":   field.Boolean,
		"date":      field.Date,
		"selection": field.Enum,
	}
	for legacy, expected := range cases {
		if got := contactFieldType(legacy); got != expected {
			t.Errorf("%s: got %s, want %s", legacy, got, expected)
		}
	}
}

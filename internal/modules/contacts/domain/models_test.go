package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestContactValidation(t *testing.T) {
	c := Contact{Kind: KindPerson, DisplayName: "Ada", Status: StatusActive}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid contact rejected: %v", err)
	}
	c.Kind = "lead"
	if err := c.Validate(); err == nil {
		t.Fatal("invalid contact kind accepted")
	}
}

func TestCustomFieldValidation(t *testing.T) {
	f := CustomFieldDefinition{Entity: "contacts.contact", Key: "tier", Label: "Tier", Type: FieldSelection, Options: []string{"gold", "silver"}}
	if err := f.Validate(); err != nil {
		t.Fatalf("valid field rejected: %v", err)
	}
	f.Key = "Tier"
	if err := f.Validate(); err == nil {
		t.Fatal("invalid custom field key accepted")
	}
}

func TestRelationshipValidation(t *testing.T) {
	r := Relationship{PersonID: uuid.New(), CompanyID: uuid.New(), RelationshipType: "employee"}
	if err := r.Validate(); err != nil {
		t.Fatalf("valid relationship rejected: %v", err)
	}
	r.CompanyID = r.PersonID
	if err := r.Validate(); err == nil {
		t.Fatal("self relationship accepted")
	}
}

package field

import (
	"fmt"
	"regexp"
	"strings"
)

type Type string

const (
	String        Type = "string"
	Text          Type = "text"
	Boolean       Type = "boolean"
	Integer       Type = "integer"
	Decimal       Type = "decimal"
	Date          Type = "date"
	DateTime      Type = "datetime"
	UUID          Type = "uuid"
	Enum          Type = "enum"
	RelationField Type = "relation"
)

type RelationKind string

const (
	ManyToOne  RelationKind = "many-to-one"
	OneToMany  RelationKind = "one-to-many"
	ManyToMany RelationKind = "many-to-many"
)

type Relation struct {
	Kind   RelationKind
	Target string
}

type Definition struct {
	Name        string
	DisplayName string
	Type        Type
	Required    bool
	ReadOnly    bool
	Description string
	Relation    *Relation
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func (d Definition) Validate() error {
	if !namePattern.MatchString(d.Name) {
		return fmt.Errorf("field name %q must be lowercase and use letters, numbers, or underscores", d.Name)
	}
	if strings.TrimSpace(d.DisplayName) == "" {
		return fmt.Errorf("field %q display name cannot be empty", d.Name)
	}
	if !validType(d.Type) {
		return fmt.Errorf("field %q has unknown type %q", d.Name, d.Type)
	}
	if d.Type == RelationField {
		if d.Relation == nil {
			return fmt.Errorf("relation field %q must define a relation", d.Name)
		}
		if d.Relation.Kind != ManyToOne && d.Relation.Kind != OneToMany && d.Relation.Kind != ManyToMany {
			return fmt.Errorf("field %q has invalid relation kind %q", d.Name, d.Relation.Kind)
		}
		if !strings.Contains(d.Relation.Target, ".") {
			return fmt.Errorf("field %q relation target %q must be namespaced", d.Name, d.Relation.Target)
		}
	} else if d.Relation != nil {
		return fmt.Errorf("field %q defines a relation for non-relation type", d.Name)
	}
	return nil
}

func validType(t Type) bool {
	switch t {
	case String, Text, Boolean, Integer, Decimal, Date, DateTime, UUID, Enum, RelationField:
		return true
	default:
		return false
	}
}

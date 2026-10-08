package domain

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
)

var (
	ErrNotFound = errors.New("customization resource not found")
	ErrConflict = errors.New("customization resource conflicts with an existing resource")
)

type FieldSource string

const (
	FieldSourceBuiltIn FieldSource = "built_in"
	FieldSourceCustom  FieldSource = "custom"
)

type RuntimeField struct {
	ID           uuid.UUID  `json:"id"`
	WorkspaceID  uuid.UUID  `json:"workspace_id"`
	Entity       string     `json:"entity"`
	Key          string     `json:"key"`
	Label        string     `json:"label"`
	Type         field.Type `json:"type"`
	Description  string     `json:"description"`
	Required     bool       `json:"required"`
	DefaultValue any        `json:"default_value,omitempty"`
	Options      []string   `json:"options"`
	Visible      bool       `json:"visible"`
	DisplayOrder int        `json:"display_order"`
	SectionID    *uuid.UUID `json:"section_id,omitempty"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// FieldPatch uses pointers to distinguish omitted properties from explicit values.
type FieldPatch struct {
	Label        *string
	Description  *string
	Required     *bool
	DefaultValue any
	SetDefault   bool
	Options      *[]string
	Visible      *bool
	DisplayOrder *int
	SectionID    *uuid.UUID
	SetSection   bool
	Active       *bool
}

type FormSection struct {
	ID           uuid.UUID `json:"id"`
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	Entity       string    `json:"entity"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Filter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value,omitempty"`
}

type SavedView struct {
	ID            uuid.UUID `json:"id"`
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	Entity        string    `json:"entity"`
	OwnerUserID   uuid.UUID `json:"owner_user_id"`
	Name          string    `json:"name"`
	Shared        bool      `json:"shared"`
	Filters       []Filter  `json:"filters"`
	Columns       []string  `json:"columns"`
	SortField     string    `json:"sort_field"`
	SortDirection string    `json:"sort_direction"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type EffectiveField struct {
	Key          string      `json:"key"`
	Label        string      `json:"label"`
	Type         field.Type  `json:"type"`
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	ReadOnly     bool        `json:"read_only"`
	Source       FieldSource `json:"source"`
	DefaultValue any         `json:"default_value,omitempty"`
	Options      []string    `json:"options"`
	Visible      bool        `json:"visible"`
	DisplayOrder int         `json:"display_order"`
	SectionID    *uuid.UUID  `json:"section_id,omitempty"`
}

type EffectiveSchema struct {
	Entity   string           `json:"entity"`
	Fields   []EffectiveField `json:"fields"`
	Sections []FormSection    `json:"sections"`
}

var fieldKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func (f RuntimeField) Validate() error {
	if !fieldKeyPattern.MatchString(f.Key) {
		return fmt.Errorf("field key is invalid")
	}
	if strings.TrimSpace(f.Label) == "" {
		return fmt.Errorf("field label is required")
	}
	if !supportedFieldType(f.Type) {
		return fmt.Errorf("field type is not supported")
	}
	if f.Type == field.Enum && len(f.Options) == 0 {
		return fmt.Errorf("selection field options are required")
	}
	if f.Type != field.Enum && len(f.Options) > 0 {
		return fmt.Errorf("only selection fields may define options")
	}
	if f.DefaultValue != nil && !ValidValue(f.Type, f.Options, f.DefaultValue) {
		return fmt.Errorf("default value does not match field type")
	}
	return nil
}

func (s FormSection) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("section name is required")
	}
	return nil
}

func (v SavedView) Validate(fieldNames map[string]struct{}) error {
	if strings.TrimSpace(v.Name) == "" {
		return fmt.Errorf("view name is required")
	}
	if v.SortDirection != "" && v.SortDirection != "asc" && v.SortDirection != "desc" {
		return fmt.Errorf("sort direction is invalid")
	}
	if v.SortField != "" {
		if _, ok := fieldNames[v.SortField]; !ok {
			return fmt.Errorf("sort field is not available")
		}
	}
	seen := map[string]struct{}{}
	for _, column := range v.Columns {
		if _, ok := fieldNames[column]; !ok {
			return fmt.Errorf("view column is not available")
		}
		if _, duplicate := seen[column]; duplicate {
			return fmt.Errorf("view columns must be unique")
		}
		seen[column] = struct{}{}
	}
	for _, filter := range v.Filters {
		if _, ok := fieldNames[filter.Field]; !ok {
			return fmt.Errorf("filter field is not available")
		}
		switch filter.Operator {
		case "eq", "neq", "contains", "in", "is_empty":
		default:
			return fmt.Errorf("filter operator is invalid")
		}
	}
	return nil
}

func ValidValue(kind field.Type, options []string, value any) bool {
	switch kind {
	case field.String, field.Text:
		_, ok := value.(string)
		return ok
	case field.Decimal:
		n, ok := value.(float64)
		return ok && !math.IsNaN(n) && !math.IsInf(n, 0)
	case field.Integer:
		n, ok := value.(float64)
		return ok && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n && math.Abs(n) <= 9007199254740991
	case field.Boolean:
		_, ok := value.(bool)
		return ok
	case field.Date:
		text, ok := value.(string)
		if !ok {
			return false
		}
		_, err := time.Parse("2006-01-02", text)
		return err == nil
	case field.Enum:
		text, ok := value.(string)
		if !ok {
			return false
		}
		return contains(options, text)
	default:
		return false
	}
}

func supportedFieldType(kind field.Type) bool {
	return kind == field.String || kind == field.Text || kind == field.Decimal || kind == field.Integer || kind == field.Boolean || kind == field.Date || kind == field.Enum
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func SortFields(fields []EffectiveField) {
	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i].DisplayOrder == fields[j].DisplayOrder {
			return fields[i].Key < fields[j].Key
		}
		return fields[i].DisplayOrder < fields[j].DisplayOrder
	})
}

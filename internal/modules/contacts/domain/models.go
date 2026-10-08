package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ContactKind string

const (
	KindPerson  ContactKind = "person"
	KindCompany ContactKind = "company"
)

type ContactStatus string

const (
	StatusActive   ContactStatus = "active"
	StatusArchived ContactStatus = "archived"
)

type Contact struct {
	ID           uuid.UUID      `json:"id"`
	WorkspaceID  uuid.UUID      `json:"workspace_id"`
	Kind         ContactKind    `json:"kind"`
	DisplayName  string         `json:"display_name"`
	Email        string         `json:"email"`
	Phone        string         `json:"phone"`
	Website      string         `json:"website"`
	Description  string         `json:"description"`
	Status       ContactStatus  `json:"status"`
	CustomValues map[string]any `json:"custom_values"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	CreatedBy    uuid.UUID      `json:"created_by"`
	UpdatedBy    uuid.UUID      `json:"updated_by"`
}

type Relationship struct {
	ID               uuid.UUID `json:"id"`
	WorkspaceID      uuid.UUID `json:"workspace_id"`
	PersonID         uuid.UUID `json:"person_id"`
	CompanyID        uuid.UUID `json:"company_id"`
	RelationshipType string    `json:"relationship_type"`
	JobTitle         string    `json:"job_title"`
	IsPrimary        bool      `json:"is_primary"`
	PersonName       string    `json:"person_name,omitempty"`
	CompanyName      string    `json:"company_name,omitempty"`
}

type Tag struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Note struct {
	ID           uuid.UUID `json:"id"`
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	ContactID    uuid.UUID `json:"contact_id"`
	AuthorUserID uuid.UUID `json:"author_user_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ActivityType string

const (
	ActivityCall     ActivityType = "call"
	ActivityEmail    ActivityType = "email"
	ActivityMeeting  ActivityType = "meeting"
	ActivityFollowUp ActivityType = "follow_up"
	ActivityTask     ActivityType = "task"
)

type ActivityStatus string

const (
	ActivityPlanned   ActivityStatus = "planned"
	ActivityCompleted ActivityStatus = "completed"
	ActivityCancelled ActivityStatus = "cancelled"
)

type Activity struct {
	ID               uuid.UUID      `json:"id"`
	WorkspaceID      uuid.UUID      `json:"workspace_id"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	ActivityType     ActivityType   `json:"activity_type"`
	RelatedContactID *uuid.UUID     `json:"related_contact_id,omitempty"`
	AssignedUserID   uuid.UUID      `json:"assigned_user_id"`
	DueAt            *time.Time     `json:"due_at,omitempty"`
	Status           ActivityStatus `json:"status"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
type Attachment struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	ContactID   uuid.UUID `json:"contact_id"`
	FileName    string    `json:"file_name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type CustomFieldType string

const (
	FieldText      CustomFieldType = "text"
	FieldNumber    CustomFieldType = "number"
	FieldBoolean   CustomFieldType = "boolean"
	FieldDate      CustomFieldType = "date"
	FieldSelection CustomFieldType = "selection"
)

type CustomFieldDefinition struct {
	ID           uuid.UUID       `json:"id"`
	WorkspaceID  uuid.UUID       `json:"workspace_id"`
	Entity       string          `json:"entity"`
	Key          string          `json:"key"`
	Label        string          `json:"label"`
	Type         CustomFieldType `json:"type"`
	Description  string          `json:"description"`
	Required     bool            `json:"required"`
	Options      []string        `json:"options"`
	DisplayOrder int             `json:"display_order"`
	Active       bool            `json:"active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type ListFilter struct {
	Search string
	Kind   ContactKind
	Status ContactStatus
	TagID  *uuid.UUID
	Page   int
	Limit  int
	Sort   string
	Desc   bool
}

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func (c Contact) Validate() error {
	if c.Kind != KindPerson && c.Kind != KindCompany {
		return fmt.Errorf("contact kind must be person or company")
	}
	if strings.TrimSpace(c.DisplayName) == "" {
		return fmt.Errorf("display name is required")
	}
	if c.Status != StatusActive && c.Status != StatusArchived {
		return ErrInvalidStatus
	}
	return nil
}
func (r Relationship) Validate() error {
	if r.PersonID == uuid.Nil || r.CompanyID == uuid.Nil {
		return fmt.Errorf("person and company are required")
	}
	if r.PersonID == r.CompanyID {
		return fmt.Errorf("a person cannot be linked to itself")
	}
	if strings.TrimSpace(r.RelationshipType) == "" {
		return fmt.Errorf("relationship type is required")
	}
	return nil
}
func (t Tag) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("tag name is required")
	}
	if strings.TrimSpace(t.Color) == "" {
		return fmt.Errorf("tag color is required")
	}
	return nil
}
func (n Note) Validate() error {
	if strings.TrimSpace(n.Content) == "" {
		return fmt.Errorf("note content is required")
	}
	if len(n.Content) > 10000 {
		return fmt.Errorf("note content is too long")
	}
	return nil
}
func (a Activity) Validate() error {
	if strings.TrimSpace(a.Title) == "" {
		return fmt.Errorf("activity title is required")
	}
	switch a.ActivityType {
	case ActivityCall, ActivityEmail, ActivityMeeting, ActivityFollowUp, ActivityTask:
	default:
		return fmt.Errorf("invalid activity type")
	}
	switch a.Status {
	case ActivityPlanned, ActivityCompleted, ActivityCancelled:
	default:
		return fmt.Errorf("invalid activity status")
	}
	return nil
}
func (f CustomFieldDefinition) Validate() error {
	if f.Entity != "contacts.contact" {
		return fmt.Errorf("custom field entity must be contacts.contact")
	}
	if !keyPattern.MatchString(f.Key) {
		return fmt.Errorf("custom field key is invalid")
	}
	if strings.TrimSpace(f.Label) == "" {
		return fmt.Errorf("custom field label is required")
	}
	switch f.Type {
	case FieldText, FieldNumber, FieldBoolean, FieldDate:
	case FieldSelection:
		if len(f.Options) == 0 {
			return fmt.Errorf("selection field options are required")
		}
	default:
		return fmt.Errorf("invalid custom field type")
	}
	return nil
}

var (
	ErrNotFound           = fmt.Errorf("contact resource not found")
	ErrDuplicate          = fmt.Errorf("contact resource already exists")
	ErrInvalidStatus      = fmt.Errorf("invalid contact status")
	ErrInvalidCustomValue = fmt.Errorf("invalid custom field value")
	ErrInactiveMember     = fmt.Errorf("assigned user is not an active workspace member")
)

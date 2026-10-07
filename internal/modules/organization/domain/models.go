package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusInactive  Status = "inactive"
	StatusInvited   Status = "invited"
	StatusSuspended Status = "suspended"
)

type Organization struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Workspace struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Slug           string
	Timezone       string
	Status         Status
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Membership struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Email       string
	DisplayName string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkspaceContext struct {
	OrganizationID uuid.UUID
	WorkspaceID    uuid.UUID
	MembershipID   uuid.UUID
	UserID         uuid.UUID
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (o Organization) Validate() error {
	if strings.TrimSpace(o.Name) == "" {
		return fmt.Errorf("organization name is required")
	}
	if !slugPattern.MatchString(o.Slug) {
		return fmt.Errorf("organization slug is invalid")
	}
	if o.Status != StatusActive && o.Status != StatusInactive {
		return ErrInvalidStatus
	}
	return nil
}

func (w Workspace) Validate() error {
	if w.OrganizationID == uuid.Nil {
		return fmt.Errorf("workspace organization is required")
	}
	if strings.TrimSpace(w.Name) == "" {
		return fmt.Errorf("workspace name is required")
	}
	if !slugPattern.MatchString(w.Slug) {
		return fmt.Errorf("workspace slug is invalid")
	}
	if strings.TrimSpace(w.Timezone) == "" {
		return fmt.Errorf("workspace timezone is required")
	}
	if w.Status != StatusActive && w.Status != StatusInactive {
		return ErrInvalidStatus
	}
	return nil
}

func (m Membership) Validate() error {
	if m.WorkspaceID == uuid.Nil || m.UserID == uuid.Nil {
		return fmt.Errorf("membership workspace and user are required")
	}
	if m.Status != StatusActive && m.Status != StatusInvited && m.Status != StatusSuspended {
		return ErrInvalidStatus
	}
	return nil
}

func Slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
		} else if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

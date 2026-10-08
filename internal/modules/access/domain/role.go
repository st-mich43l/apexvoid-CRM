package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[_-][a-z0-9]+)*$`)

type Role struct {
	ID          uuid.UUID
	WorkspaceID *uuid.UUID
	Name        string
	DisplayName string
	Description string
	System      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r Role) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("role name is required")
	}
	if !namePattern.MatchString(r.Name) {
		return fmt.Errorf("role name must be lowercase and machine friendly")
	}
	if r.Name == "administrator" || r.Name == "workspace_administrator" {
		validSystemIdentity := r.System && ((r.Name == "administrator" && r.WorkspaceID == nil) || (r.Name == "workspace_administrator" && r.WorkspaceID != nil))
		if !validSystemIdentity {
			return ErrReservedRoleName
		}
	}
	if strings.TrimSpace(r.DisplayName) == "" {
		return fmt.Errorf("role display name is required")
	}
	return nil
}

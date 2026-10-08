package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestReservedAdministratorNamesRequireProtectedIdentity(t *testing.T) {
	workspaceID := uuid.New()
	for _, test := range []struct {
		name        string
		workspaceID *uuid.UUID
		system      bool
	}{
		{name: "administrator"},
		{name: "administrator", system: true, workspaceID: &workspaceID},
		{name: "workspace_administrator", workspaceID: &workspaceID},
		{name: "workspace_administrator", system: true},
	} {
		role := Role{Name: test.name, DisplayName: "Role", WorkspaceID: test.workspaceID, System: test.system}
		if err := role.Validate(); err != ErrReservedRoleName {
			t.Fatalf("Validate(%+v) returned %v, want ErrReservedRoleName", test, err)
		}
	}
}

func TestProtectedAdministratorIdentitiesValidate(t *testing.T) {
	workspaceID := uuid.New()
	for _, role := range []Role{
		{Name: "administrator", DisplayName: "Administrator", System: true},
		{Name: "workspace_administrator", DisplayName: "Workspace Administrator", WorkspaceID: &workspaceID, System: true},
	} {
		if err := role.Validate(); err != nil {
			t.Fatalf("protected identity %q rejected: %v", role.Name, err)
		}
	}
}

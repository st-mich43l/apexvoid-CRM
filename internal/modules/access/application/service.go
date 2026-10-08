package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/access/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Service struct {
	repository   domain.Repository
	permissions  *permission.Registry
	users        usersapi.UserReader
	transactions *database.TxManager
}

type Dependencies struct {
	Repository   domain.Repository
	Permissions  *permission.Registry
	Users        usersapi.UserReader
	Transactions *database.TxManager
}

func NewService(dependencies Dependencies) *Service {
	return &Service{repository: dependencies.Repository, permissions: dependencies.Permissions, users: dependencies.Users, transactions: dependencies.Transactions}
}

type CreateRoleInput struct{ Name, DisplayName, Description string }
type UpdateRoleInput struct{ DisplayName, Description *string }

func (s *Service) EnsureAdministrator(ctx context.Context) (domain.Role, error) {
	return s.repository.EnsureAdministrator(ctx)
}

func (s *Service) CreateRole(ctx context.Context, input CreateRoleInput) (domain.Role, error) {
	role := domain.Role{ID: uuid.New(), Name: strings.TrimSpace(input.Name), DisplayName: strings.TrimSpace(input.DisplayName), Description: strings.TrimSpace(input.Description), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := role.Validate(); err != nil {
		return domain.Role{}, err
	}
	if err := s.repository.CreateRole(ctx, &role); err != nil {
		return domain.Role{}, err
	}
	return role, nil
}

func (s *Service) createWorkspaceRole(ctx context.Context, workspaceID uuid.UUID, input CreateRoleInput) (domain.Role, error) {
	role := domain.Role{ID: uuid.New(), WorkspaceID: &workspaceID, Name: strings.TrimSpace(input.Name), DisplayName: strings.TrimSpace(input.DisplayName), Description: strings.TrimSpace(input.Description), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := role.Validate(); err != nil {
		return domain.Role{}, err
	}
	if err := s.repository.CreateRole(ctx, &role); err != nil {
		return domain.Role{}, err
	}
	return role, nil
}

func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.repository.ListRoles(ctx)
}

func (s *Service) ListWorkspaceRoles(ctx context.Context, workspaceID uuid.UUID) ([]organizationapi.WorkspaceRole, error) {
	roles, err := s.repository.ListWorkspaceRoles(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]organizationapi.WorkspaceRole, 0, len(roles))
	for _, role := range roles {
		permissions, permissionErr := s.repository.RolePermissions(ctx, role.ID)
		if permissionErr != nil {
			return nil, permissionErr
		}
		result = append(result, organizationapi.WorkspaceRole{ID: role.ID, WorkspaceID: role.WorkspaceID, Name: role.Name, DisplayName: role.DisplayName, Description: role.Description, System: role.System, Permissions: permissions})
	}
	return result, nil
}

func (s *Service) CreateWorkspaceRole(ctx context.Context, workspaceID uuid.UUID, name, displayName, description string) (organizationapi.WorkspaceRole, error) {
	role, err := s.createWorkspaceRole(ctx, workspaceID, CreateRoleInput{Name: name, DisplayName: displayName, Description: description})
	if err != nil {
		return organizationapi.WorkspaceRole{}, err
	}
	return organizationapi.WorkspaceRole{ID: role.ID, WorkspaceID: role.WorkspaceID, Name: role.Name, DisplayName: role.DisplayName, Description: role.Description, System: role.System, Permissions: []string{}}, nil
}
func (s *Service) GetRole(ctx context.Context, id uuid.UUID) (domain.Role, error) {
	role, err := s.repository.FindRole(ctx, id)
	if err != nil {
		return domain.Role{}, err
	}
	return *role, nil
}

func (s *Service) UpdateRole(ctx context.Context, id uuid.UUID, input UpdateRoleInput) (domain.Role, error) {
	role, err := s.repository.FindRole(ctx, id)
	if err != nil {
		return domain.Role{}, err
	}
	if role.System {
		if input.DisplayName != nil && strings.TrimSpace(*input.DisplayName) != role.DisplayName {
			return domain.Role{}, domain.ErrSystemRole
		}
	}
	if input.DisplayName != nil {
		role.DisplayName = strings.TrimSpace(*input.DisplayName)
	}
	if input.Description != nil {
		role.Description = strings.TrimSpace(*input.Description)
	}
	role.UpdatedAt = time.Now().UTC()
	if err := role.Validate(); err != nil {
		return domain.Role{}, err
	}
	if err := s.repository.UpdateRole(ctx, role); err != nil {
		return domain.Role{}, err
	}
	return *role, nil
}

func (s *Service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	role, err := s.repository.FindRole(ctx, id)
	if err != nil {
		return err
	}
	if role.System || role.Name == "administrator" {
		return domain.ErrSystemRole
	}
	return s.repository.DeleteRole(ctx, id)
}

func (s *Service) Permissions(ctx context.Context) []permission.Definition {
	if s.permissions == nil {
		return nil
	}
	return s.permissions.List()
}

func (s *Service) RolePermissions(ctx context.Context, id uuid.UUID) ([]string, error) {
	return s.repository.RolePermissions(ctx, id)
}

func (s *Service) ReplaceRolePermissions(ctx context.Context, id uuid.UUID, names []string) error {
	role, err := s.repository.FindRole(ctx, id)
	if err != nil {
		return err
	}
	if role.System {
		return domain.ErrSystemRole
	}
	seen := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if _, ok := seen[name]; ok {
			continue
		}
		if s.permissions == nil || !s.permissions.Contains(name) {
			return domain.ErrUnknownPermission
		}
		definition, _ := s.permissions.Get(name)
		if definition.Scope != permission.ScopePlatform {
			return domain.ErrPermissionScope
		}
		seen[name] = struct{}{}
	}
	unique := make([]string, 0, len(seen))
	for name := range seen {
		unique = append(unique, name)
	}
	return s.withTransaction(ctx, func(txCtx context.Context) error { return s.repository.ReplaceRolePermissions(txCtx, id, unique) })
}

func (s *Service) ReplaceWorkspaceRolePermissions(ctx context.Context, id, workspaceID uuid.UUID, names []string) error {
	role, err := s.repository.FindRole(ctx, id)
	if err != nil {
		return err
	}
	if role.WorkspaceID == nil || *role.WorkspaceID != workspaceID {
		return domain.ErrWorkspaceRole
	}
	seen := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if _, ok := seen[name]; ok {
			continue
		}
		if s.permissions == nil || !s.permissions.Contains(name) {
			return domain.ErrUnknownPermission
		}
		definition, _ := s.permissions.Get(name)
		if definition.Scope != permission.ScopeWorkspace {
			return domain.ErrPermissionScope
		}
		seen[name] = struct{}{}
	}
	unique := make([]string, 0, len(seen))
	for name := range seen {
		unique = append(unique, name)
	}
	return s.withTransaction(ctx, func(txCtx context.Context) error { return s.repository.ReplaceRolePermissions(txCtx, id, unique) })
}

func (s *Service) UserRoleIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return s.repository.UserRoleIDs(ctx, userID)
}

func (s *Service) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	if _, err := s.users.FindActiveByID(ctx, userID); err != nil {
		return err
	}
	seen := map[uuid.UUID]struct{}{}
	unique := make([]uuid.UUID, 0, len(roleIDs))
	for _, id := range roleIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		role, err := s.repository.FindRole(ctx, id)
		if err != nil {
			return err
		}
		if role.WorkspaceID != nil {
			return domain.ErrAssignmentNotAllowed
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return s.withTransaction(ctx, func(txCtx context.Context) error { return s.repository.ReplaceUserRoles(txCtx, userID, unique) })
}

func (s *Service) EffectivePermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	permissions, administrator, err := s.repository.EffectivePermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0, len(permissions))
	if s.permissions != nil {
		for _, name := range permissions {
			if definition, ok := s.permissions.Get(name); ok && definition.Scope == permission.ScopePlatform {
				filtered = append(filtered, name)
			}
		}
	}
	permissions = filtered
	if administrator && s.permissions != nil {
		for _, definition := range s.permissions.List() {
			if definition.Scope == permission.ScopePlatform {
				permissions = append(permissions, definition.Name)
			}
		}
	}
	set := map[string]struct{}{}
	for _, name := range permissions {
		set[name] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	return result, nil
}

func (s *Service) Can(ctx context.Context, userID uuid.UUID, name string) (bool, error) {
	if _, err := s.users.FindActiveByID(ctx, userID); err != nil {
		return false, err
	}
	permissions, err := s.EffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, permission := range permissions {
		if permission == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) EffectivePermissionsForWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) ([]string, error) {
	platformPermissions, workspacePermissions, administrator, workspaceAdministrator, err := s.repository.EffectivePermissionsForWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	permissions := make([]string, 0, len(platformPermissions)+len(workspacePermissions))
	if s.permissions != nil {
		for _, name := range platformPermissions {
			if definition, ok := s.permissions.Get(name); ok && definition.Scope == permission.ScopePlatform {
				permissions = append(permissions, name)
			}
		}
		for _, name := range workspacePermissions {
			if definition, ok := s.permissions.Get(name); ok && definition.Scope == permission.ScopeWorkspace {
				permissions = append(permissions, name)
			}
		}
	}
	if s.permissions != nil {
		for _, definition := range s.permissions.List() {
			if administrator && definition.Scope == permission.ScopePlatform || workspaceAdministrator && definition.Scope == permission.ScopeWorkspace {
				permissions = append(permissions, definition.Name)
			}
		}
	}
	set := map[string]struct{}{}
	for _, name := range permissions {
		set[name] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	return result, nil
}

func (s *Service) CanInWorkspace(ctx context.Context, userID, workspaceID uuid.UUID, name string) (bool, error) {
	permissions, err := s.EffectivePermissionsForWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return false, err
	}
	for _, item := range permissions {
		if item == name {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) CountActiveWorkspaceAdministrators(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	return s.repository.CountActiveWorkspaceAdministrators(ctx, workspaceID)
}

func (s *Service) IsWorkspaceAdministrator(ctx context.Context, membershipID, workspaceID uuid.UUID) (bool, error) {
	return s.repository.IsWorkspaceAdministrator(ctx, membershipID, workspaceID)
}

func (s *Service) EnsureWorkspaceAdministrator(ctx context.Context, workspaceID uuid.UUID) error {
	role, err := s.repository.FindRoleByWorkspaceAndName(ctx, &workspaceID, "workspace_administrator")
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if errors.Is(err, domain.ErrNotFound) {
		now := time.Now().UTC()
		role = &domain.Role{ID: uuid.New(), WorkspaceID: &workspaceID, Name: "workspace_administrator", DisplayName: "Workspace Administrator", Description: "Manages this workspace and its members.", System: true, CreatedAt: now, UpdatedAt: now}
		createErr := s.repository.CreateRole(ctx, role)
		if createErr != nil && !errors.Is(createErr, domain.ErrDuplicateRole) {
			return createErr
		}
		if createErr != nil {
			role, err = s.repository.FindRoleByWorkspaceAndName(ctx, &workspaceID, "workspace_administrator")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) AssignWorkspaceAdministrator(ctx context.Context, membershipID, workspaceID uuid.UUID) error {
	role, err := s.repository.FindRoleByWorkspaceAndName(ctx, &workspaceID, "workspace_administrator")
	if err != nil {
		return err
	}
	return s.repository.ReplaceMembershipRoles(ctx, membershipID, []uuid.UUID{role.ID})
}

func (s *Service) MembershipRoleIDs(ctx context.Context, membershipID uuid.UUID) ([]uuid.UUID, error) {
	return s.repository.MembershipRoleIDs(ctx, membershipID)
}

func (s *Service) ReplaceMembershipRoles(ctx context.Context, membershipID, workspaceID uuid.UUID, roleIDs []uuid.UUID) error {
	seen := map[uuid.UUID]struct{}{}
	willBeAdministrator := false
	for _, roleID := range roleIDs {
		if _, ok := seen[roleID]; ok {
			continue
		}
		role, err := s.repository.FindRole(ctx, roleID)
		if err != nil {
			return err
		}
		if role.WorkspaceID == nil || *role.WorkspaceID != workspaceID {
			return domain.ErrWorkspaceRole
		}
		if role.Name == "workspace_administrator" {
			willBeAdministrator = true
		}
		seen[roleID] = struct{}{}
	}
	unique := make([]uuid.UUID, 0, len(seen))
	for roleID := range seen {
		unique = append(unique, roleID)
	}
	return s.withTransaction(ctx, func(txCtx context.Context) error {
		isAdministrator, err := s.repository.IsWorkspaceAdministrator(txCtx, membershipID, workspaceID)
		if err != nil {
			return err
		}
		if isAdministrator && !willBeAdministrator {
			count, err := s.repository.CountActiveWorkspaceAdministrators(txCtx, workspaceID)
			if err != nil {
				return err
			}
			if count <= 1 {
				return domain.ErrLastWorkspaceAdministrator
			}
		}
		return s.repository.ReplaceMembershipRoles(txCtx, membershipID, unique)
	})
}

func (s *Service) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if s.transactions == nil {
		return fn(ctx)
	}
	return s.transactions.WithTransaction(ctx, fn)
}

var _ usersapi.Authorizer = (*Service)(nil)

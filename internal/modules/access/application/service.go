package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/access/domain"
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

func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.repository.ListRoles(ctx)
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
		if _, err := s.repository.FindRole(ctx, id); err != nil {
			return err
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
	if administrator && s.permissions != nil {
		for _, definition := range s.permissions.List() {
			permissions = append(permissions, definition.Name)
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

func (s *Service) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if s.transactions == nil {
		return fn(ctx)
	}
	return s.transactions.WithTransaction(ctx, fn)
}

var _ usersapi.Authorizer = (*Service)(nil)

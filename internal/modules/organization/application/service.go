package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Service struct {
	repository   domain.Repository
	transactions *database.TxManager
	access       organizationapi.WorkspaceAccess
	users        usersapi.UserReader
	directory    usersapi.UserDirectory
}

type Dependencies struct {
	Repository   domain.Repository
	Transactions *database.TxManager
	Access       organizationapi.WorkspaceAccess
	Users        usersapi.UserReader
	Directory    usersapi.UserDirectory
}

func NewService(dependencies Dependencies) *Service {
	return &Service{repository: dependencies.Repository, transactions: dependencies.Transactions, access: dependencies.Access, users: dependencies.Users, directory: dependencies.Directory}
}

func (s *Service) SetAccess(access organizationapi.WorkspaceAccess) { s.access = access }

type SetupInput struct {
	OrganizationName string
	WorkspaceName    string
	Timezone         string
}

type SetupStatus struct {
	Required     bool `json:"required"`
	Organization bool `json:"organization"`
	Workspace    bool `json:"workspace"`
}

func (s *Service) Status(ctx context.Context, userID uuid.UUID) (SetupStatus, error) {
	count, err := s.repository.CountOrganizations(ctx)
	if err != nil {
		return SetupStatus{}, err
	}
	if count == 0 {
		return SetupStatus{Required: true}, nil
	}
	workspaces, err := s.repository.ListWorkspaces(ctx, userID)
	if err != nil {
		return SetupStatus{}, err
	}
	return SetupStatus{Required: false, Organization: true, Workspace: len(workspaces) > 0}, nil
}

func (s *Service) Setup(ctx context.Context, userID uuid.UUID, input SetupInput) (domain.Organization, domain.Workspace, domain.Membership, error) {
	if s.access == nil {
		return domain.Organization{}, domain.Workspace{}, domain.Membership{}, errors.New("organization access provisioning is not configured")
	}
	if strings.TrimSpace(input.OrganizationName) == "" {
		return domain.Organization{}, domain.Workspace{}, domain.Membership{}, errors.New("organization name is required")
	}
	workspaceName := strings.TrimSpace(input.WorkspaceName)
	if workspaceName == "" {
		workspaceName = "Main Workspace"
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	now := time.Now().UTC()
	organization := domain.Organization{ID: uuid.New(), Name: strings.TrimSpace(input.OrganizationName), Slug: domain.Slugify(input.OrganizationName), Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	workspace := domain.Workspace{ID: uuid.New(), OrganizationID: organization.ID, Name: workspaceName, Slug: domain.Slugify(workspaceName), Timezone: timezone, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	membership := domain.Membership{ID: uuid.New(), WorkspaceID: workspace.ID, UserID: userID, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := organization.Validate(); err != nil {
		return domain.Organization{}, domain.Workspace{}, domain.Membership{}, err
	}
	if err := workspace.Validate(); err != nil {
		return domain.Organization{}, domain.Workspace{}, domain.Membership{}, err
	}
	if err := membership.Validate(); err != nil {
		return domain.Organization{}, domain.Workspace{}, domain.Membership{}, err
	}
	err := s.withTransaction(ctx, func(txCtx context.Context) error {
		count, err := s.repository.CountOrganizations(txCtx)
		if err != nil {
			return err
		}
		if count != 0 {
			return domain.ErrSetupComplete
		}
		if err := s.repository.CreateOrganization(txCtx, &organization); err != nil {
			return err
		}
		if err := s.repository.CreateWorkspace(txCtx, &workspace); err != nil {
			return err
		}
		if err := s.repository.CreateMembership(txCtx, &membership); err != nil {
			return err
		}
		if err := s.access.EnsureWorkspaceAdministrator(txCtx, workspace.ID); err != nil {
			return err
		}
		return s.access.AssignWorkspaceAdministrator(txCtx, membership.ID, workspace.ID)
	})
	if err != nil {
		return domain.Organization{}, domain.Workspace{}, domain.Membership{}, err
	}
	return organization, workspace, membership, nil
}

func (s *Service) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	return s.repository.ListWorkspaces(ctx, userID)
}

func (s *Service) ResolveWorkspaceContext(ctx context.Context, userID, workspaceID uuid.UUID) (domain.WorkspaceContext, error) {
	workspace, err := s.repository.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return domain.WorkspaceContext{}, err
	}
	if workspace.Status != domain.StatusActive {
		return domain.WorkspaceContext{}, domain.ErrWorkspaceInactive
	}
	organization, err := s.repository.GetOrganization(ctx, workspace.OrganizationID)
	if err != nil {
		return domain.WorkspaceContext{}, err
	}
	if organization.Status != domain.StatusActive {
		return domain.WorkspaceContext{}, domain.ErrOrganizationInactive
	}
	membership, err := s.repository.FindMembership(ctx, userID, workspaceID)
	if err != nil {
		return domain.WorkspaceContext{}, err
	}
	if membership.Status != domain.StatusActive {
		return domain.WorkspaceContext{}, domain.ErrMembershipSuspended
	}
	return domain.WorkspaceContext{OrganizationID: organization.ID, WorkspaceID: workspace.ID, MembershipID: membership.ID, UserID: userID}, nil
}

func (s *Service) ResolveDefaultWorkspaceContext(ctx context.Context, userID uuid.UUID) (domain.WorkspaceContext, error) {
	workspaces, err := s.repository.ListWorkspaces(ctx, userID)
	if err != nil {
		return domain.WorkspaceContext{}, err
	}
	if len(workspaces) != 1 {
		return domain.WorkspaceContext{}, domain.ErrNotFound
	}
	return s.ResolveWorkspaceContext(ctx, userID, workspaces[0].ID)
}

func (s *Service) GetOrganization(ctx context.Context, workspaceID uuid.UUID) (domain.Organization, error) {
	organization, err := s.repository.GetOrganizationForWorkspace(ctx, workspaceID)
	if err != nil {
		return domain.Organization{}, err
	}
	return *organization, nil
}

func (s *Service) UpdateOrganization(ctx context.Context, workspaceID uuid.UUID, name *string) (domain.Organization, error) {
	organization, err := s.repository.GetOrganizationForWorkspace(ctx, workspaceID)
	if err != nil {
		return domain.Organization{}, err
	}
	if name != nil {
		organization.Name = strings.TrimSpace(*name)
	}
	organization.UpdatedAt = time.Now().UTC()
	if err := organization.Validate(); err != nil {
		return domain.Organization{}, err
	}
	if err := s.repository.UpdateOrganization(ctx, organization); err != nil {
		return domain.Organization{}, err
	}
	return *organization, nil
}

func (s *Service) CreateWorkspace(ctx context.Context, current domain.WorkspaceContext, name, timezone string) (domain.Workspace, error) {
	if s.access == nil {
		return domain.Workspace{}, errors.New("organization access provisioning is not configured")
	}
	organization, err := s.repository.GetOrganization(ctx, current.OrganizationID)
	if err != nil {
		return domain.Workspace{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Workspace{}, errors.New("workspace name is required")
	}
	if strings.TrimSpace(timezone) == "" {
		timezone = "UTC"
	}
	now := time.Now().UTC()
	workspace := domain.Workspace{ID: uuid.New(), OrganizationID: organization.ID, Name: name, Slug: domain.Slugify(name), Timezone: strings.TrimSpace(timezone), Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := workspace.Validate(); err != nil {
		return domain.Workspace{}, err
	}
	membership := domain.Membership{ID: uuid.New(), WorkspaceID: workspace.ID, UserID: current.UserID, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	err = s.withTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repository.CreateWorkspace(txCtx, &workspace); err != nil {
			return err
		}
		if err := s.repository.CreateMembership(txCtx, &membership); err != nil {
			return err
		}
		if err := s.access.EnsureWorkspaceAdministrator(txCtx, workspace.ID); err != nil {
			return err
		}
		return s.access.AssignWorkspaceAdministrator(txCtx, membership.ID, workspace.ID)
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	return workspace, nil
}

func (s *Service) GetWorkspace(ctx context.Context, id uuid.UUID) (domain.Workspace, error) {
	workspace, err := s.repository.GetWorkspace(ctx, id)
	if err != nil {
		return domain.Workspace{}, err
	}
	return *workspace, nil
}

func (s *Service) UpdateWorkspace(ctx context.Context, id uuid.UUID, name, timezone *string) (domain.Workspace, error) {
	workspace, err := s.repository.GetWorkspace(ctx, id)
	if err != nil {
		return domain.Workspace{}, err
	}
	if name != nil {
		workspace.Name = strings.TrimSpace(*name)
	}
	if timezone != nil {
		workspace.Timezone = strings.TrimSpace(*timezone)
	}
	workspace.UpdatedAt = time.Now().UTC()
	if err := workspace.Validate(); err != nil {
		return domain.Workspace{}, err
	}
	if err := s.repository.UpdateWorkspace(ctx, workspace); err != nil {
		return domain.Workspace{}, err
	}
	return *workspace, nil
}

func (s *Service) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error) {
	return s.repository.ListMembers(ctx, workspaceID)
}

func (s *Service) AddMember(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Membership, error) {
	if s.users != nil {
		if _, err := s.users.FindActiveByID(ctx, userID); err != nil {
			return domain.Membership{}, err
		}
	}
	now := time.Now().UTC()
	membership := domain.Membership{ID: uuid.New(), WorkspaceID: workspaceID, UserID: userID, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := membership.Validate(); err != nil {
		return domain.Membership{}, err
	}
	if err := s.repository.CreateMembership(ctx, &membership); err != nil {
		return domain.Membership{}, err
	}
	result, err := s.repository.FindMembershipByID(ctx, membership.ID)
	if err != nil {
		return domain.Membership{}, err
	}
	return *result, nil
}

func (s *Service) ListUserCandidates(ctx context.Context) ([]usersapi.UserSummary, error) {
	if s.directory == nil {
		return nil, errors.New("user directory is not configured")
	}
	return s.directory.ListActive(ctx)
}

func (s *Service) UpdateMembership(ctx context.Context, workspaceID, id uuid.UUID, status domain.Status) (domain.Membership, error) {
	membership, err := s.repository.FindMembershipByID(ctx, id)
	if err != nil {
		return domain.Membership{}, err
	}
	if membership.WorkspaceID != workspaceID {
		return domain.Membership{}, domain.ErrMembershipNotFound
	}
	membership.Status = status
	membership.UpdatedAt = time.Now().UTC()
	if err := membership.Validate(); err != nil {
		return domain.Membership{}, err
	}
	if err := s.repository.UpdateMembership(ctx, membership); err != nil {
		return domain.Membership{}, err
	}
	return *membership, nil
}

func (s *Service) RemoveMember(ctx context.Context, workspaceID, id uuid.UUID) error {
	membership, err := s.repository.FindMembershipByID(ctx, id)
	if err != nil {
		return err
	}
	if membership.WorkspaceID != workspaceID {
		return domain.ErrMembershipNotFound
	}
	return s.repository.DeleteMembership(ctx, id)
}

func (s *Service) EnsureMembership(ctx context.Context, workspaceID, id uuid.UUID) error {
	membership, err := s.repository.FindMembershipByID(ctx, id)
	if err != nil {
		return err
	}
	if membership.WorkspaceID != workspaceID {
		return domain.ErrMembershipNotFound
	}
	return nil
}

func (s *Service) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if s.transactions == nil {
		return fn(ctx)
	}
	return s.transactions.WithTransaction(ctx, fn)
}

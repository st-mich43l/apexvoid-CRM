package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
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
	events       *event.Bus
	logger       *slog.Logger
}

type Dependencies struct {
	Repository   domain.Repository
	Transactions *database.TxManager
	Access       organizationapi.WorkspaceAccess
	Users        usersapi.UserReader
	Directory    usersapi.UserDirectory
	Events       *event.Bus
	Logger       *slog.Logger
}

func NewService(dependencies Dependencies) *Service {
	logger := dependencies.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repository: dependencies.Repository, transactions: dependencies.Transactions, access: dependencies.Access, users: dependencies.Users, directory: dependencies.Directory, events: dependencies.Events, logger: logger}
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
	s.publish(ctx, "organization.organization.created", domain.OrganizationCreated{OrganizationID: organization.ID})
	s.publish(ctx, "workspace.workspace.created", domain.WorkspaceCreated{WorkspaceID: workspace.ID, OrganizationID: organization.ID})
	s.publish(ctx, "workspace.member.added", domain.MemberAdded{WorkspaceID: workspace.ID, MembershipID: membership.ID, UserID: membership.UserID})
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
	if err := s.withTransaction(ctx, func(txCtx context.Context) error { return s.repository.UpdateOrganization(txCtx, organization) }); err != nil {
		return domain.Organization{}, err
	}
	s.publish(ctx, "organization.organization.updated", domain.OrganizationUpdated{OrganizationID: organization.ID})
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
	s.publish(ctx, "workspace.workspace.created", domain.WorkspaceCreated{WorkspaceID: workspace.ID, OrganizationID: workspace.OrganizationID})
	s.publish(ctx, "workspace.member.added", domain.MemberAdded{WorkspaceID: workspace.ID, MembershipID: membership.ID, UserID: membership.UserID})
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
	if err := s.withTransaction(ctx, func(txCtx context.Context) error { return s.repository.UpdateWorkspace(txCtx, workspace) }); err != nil {
		return domain.Workspace{}, err
	}
	s.publish(ctx, "workspace.workspace.updated", domain.WorkspaceUpdated{WorkspaceID: workspace.ID, OrganizationID: workspace.OrganizationID})
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
	if err := s.withTransaction(ctx, func(txCtx context.Context) error { return s.repository.CreateMembership(txCtx, &membership) }); err != nil {
		return domain.Membership{}, err
	}
	result, err := s.repository.FindMembershipByID(ctx, membership.ID)
	if err != nil {
		return domain.Membership{}, err
	}
	s.publish(ctx, "workspace.member.added", domain.MemberAdded{WorkspaceID: result.WorkspaceID, MembershipID: result.ID, UserID: result.UserID})
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
	if err := s.withTransaction(ctx, func(txCtx context.Context) error {
		if s.access != nil && status != domain.StatusActive {
			if err := s.access.LockWorkspaceAdministratorState(txCtx, workspaceID); err != nil {
				return err
			}
			isAdministrator, err := s.access.IsWorkspaceAdministrator(txCtx, id, workspaceID)
			if err != nil {
				return err
			}
			if isAdministrator {
				count, err := s.access.CountActiveWorkspaceAdministrators(txCtx, workspaceID)
				if err != nil {
					return err
				}
				if count <= 1 {
					return domain.ErrLastWorkspaceAdministrator
				}
			}
		}
		return s.repository.UpdateMembership(txCtx, membership)
	}); err != nil {
		return domain.Membership{}, err
	}
	s.publish(ctx, "workspace.member.updated", domain.MemberUpdated{WorkspaceID: membership.WorkspaceID, MembershipID: membership.ID, UserID: membership.UserID})
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
	if err := s.withTransaction(ctx, func(txCtx context.Context) error {
		if s.access != nil && membership.Status == domain.StatusActive {
			if err := s.access.LockWorkspaceAdministratorState(txCtx, workspaceID); err != nil {
				return err
			}
			isAdministrator, err := s.access.IsWorkspaceAdministrator(txCtx, id, workspaceID)
			if err != nil {
				return err
			}
			if isAdministrator {
				count, err := s.access.CountActiveWorkspaceAdministrators(txCtx, workspaceID)
				if err != nil {
					return err
				}
				if count <= 1 {
					return domain.ErrLastWorkspaceAdministrator
				}
			}
		}
		return s.repository.DeleteMembership(txCtx, id)
	}); err != nil {
		return err
	}
	s.publish(ctx, "workspace.member.removed", domain.MemberRemoved{WorkspaceID: membership.WorkspaceID, MembershipID: membership.ID, UserID: membership.UserID})
	return nil
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

func (s *Service) publish(ctx context.Context, name string, payload any) {
	if s.events == nil {
		return
	}
	database.AfterCommit(ctx, func(ctx context.Context) {
		var err error
		switch typed := payload.(type) {
		case domain.OrganizationCreated:
			err = event.Publish(s.events, ctx, name, typed)
		case domain.OrganizationUpdated:
			err = event.Publish(s.events, ctx, name, typed)
		case domain.WorkspaceCreated:
			err = event.Publish(s.events, ctx, name, typed)
		case domain.WorkspaceUpdated:
			err = event.Publish(s.events, ctx, name, typed)
		case domain.MemberAdded:
			err = event.Publish(s.events, ctx, name, typed)
		case domain.MemberUpdated:
			err = event.Publish(s.events, ctx, name, typed)
		case domain.MemberRemoved:
			err = event.Publish(s.events, ctx, name, typed)
		default:
			err = errors.New("unsupported organization event payload")
		}
		if err != nil {
			s.logger.Error("post-commit event publication failed", "event", name, "error", err)
		}
	})
}

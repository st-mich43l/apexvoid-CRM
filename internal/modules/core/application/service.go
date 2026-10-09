package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	frameworkerrors "github.com/st-mich43l/apexvoid-CRM/internal/framework/errors"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/api"
)

type Service struct {
	reader     api.MetadataReader
	authorizer api.ApplicationAuthorizer
}

var _ api.FrameworkReader = (*Service)(nil)

func NewService(reader api.MetadataReader, authorizer api.ApplicationAuthorizer) *Service {
	return &Service{reader: reader, authorizer: authorizer}
}

func (s *Service) Applications(ctx context.Context, userID, workspaceID uuid.UUID) ([]api.DiscoveredApplication, error) {
	if s.authorizer == nil {
		return nil, fmt.Errorf("application authorization is not configured")
	}
	snapshot := s.reader.Snapshot()
	permissions := make(map[string]permission.Definition, len(snapshot.Permissions))
	for _, definition := range snapshot.Permissions {
		permissions[definition.Name] = definition
	}
	result := make([]api.DiscoveredApplication, 0, len(snapshot.Applications))
	for _, definition := range snapshot.Applications {
		entryAuthorized, err := s.authorize(ctx, userID, workspaceID, permissions, definition.Access.Entry)
		if err != nil {
			return nil, fmt.Errorf("authorize application %q entry: %w", definition.ID, err)
		}
		settingsAuthorized := false
		if definition.Access.Settings != nil {
			settingsAuthorized, err = s.authorize(ctx, userID, workspaceID, permissions, *definition.Access.Settings)
			if err != nil {
				return nil, fmt.Errorf("authorize application %q settings: %w", definition.ID, err)
			}
		}
		result = append(result, api.DiscoveredApplication{Descriptor: definition, EntryAuthorized: entryAuthorized, SettingsAuthorized: settingsAuthorized})
	}
	return result, nil
}

func (s *Service) authorize(ctx context.Context, userID, workspaceID uuid.UUID, definitions map[string]permission.Definition, policy frameworkapplication.PermissionPolicy) (bool, error) {
	for _, name := range policy.Permissions {
		definition, ok := definitions[name]
		if !ok {
			return false, fmt.Errorf("permission %q is not registered", name)
		}
		var allowed bool
		var err error
		switch definition.Scope {
		case permission.ScopePlatform:
			allowed, err = s.authorizer.Can(ctx, userID, name)
		case permission.ScopeWorkspace:
			allowed, err = s.authorizer.CanInWorkspace(ctx, userID, workspaceID, name)
		default:
			return false, fmt.Errorf("permission %q has unsupported scope %q", name, definition.Scope)
		}
		if err != nil {
			return false, err
		}
		if policy.Match == frameworkapplication.PermissionMatchAny && allowed {
			return true, nil
		}
		if policy.Match == frameworkapplication.PermissionMatchAll && !allowed {
			return false, nil
		}
	}
	return policy.Match == frameworkapplication.PermissionMatchAll, nil
}

func (s *Service) Modules(_ context.Context) []metadata.ModuleMetadata {
	return metadata.ModuleDescriptors(s.reader.Snapshot().Modules)
}

func (s *Service) Entities(_ context.Context) []entity.Definition {
	return s.reader.Snapshot().Entities
}

func (s *Service) Permissions(_ context.Context) []metadata.PermissionMetadata {
	snapshot := s.reader.Snapshot()
	result := make([]metadata.PermissionMetadata, 0, len(snapshot.Permissions))
	for _, item := range snapshot.Permissions {
		result = append(result, metadata.PermissionMetadata{Name: item.Name, Module: item.Module, Scope: item.Scope, DisplayName: item.DisplayName, Description: item.Description})
	}
	return result
}

func (s *Service) Entity(_ context.Context, name string) (entity.Definition, error) {
	definition, ok := s.reader.Entity(name)
	if !ok {
		return entity.Definition{}, frameworkerrors.NotFoundError("framework entity")
	}
	return definition, nil
}

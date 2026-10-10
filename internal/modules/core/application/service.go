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
	external   *ExternalStore
}

var _ api.FrameworkReader = (*Service)(nil)

func NewService(reader api.MetadataReader, authorizer api.ApplicationAuthorizer, external ...*ExternalStore) *Service {
	service := &Service{reader: reader, authorizer: authorizer}
	if len(external) > 0 {
		service.external = external[0]
	}
	return service
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
	if s.external != nil {
		external, err := s.external.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list external applications: %w", err)
		}
		for _, registered := range external {
			enabled, enableErr := s.external.EnabledInWorkspace(ctx, registered.ID, workspaceID)
			if enableErr != nil {
				return nil, fmt.Errorf("resolve application %q availability: %w", registered.ID, enableErr)
			}
			if !enabled {
				continue
			}
			definition := s.external.Descriptor(registered)
			entryAuthorized, authorizeErr := s.authorize(ctx, userID, workspaceID, permissions, definition.Access.Entry)
			if authorizeErr != nil {
				return nil, fmt.Errorf("authorize external application %q entry: %w", definition.ID, authorizeErr)
			}
			settingsAuthorized := false
			if definition.Access.Settings != nil {
				settingsAuthorized, authorizeErr = s.authorize(ctx, userID, workspaceID, permissions, *definition.Access.Settings)
				if authorizeErr != nil {
					return nil, authorizeErr
				}
			}
			result = append(result, api.DiscoveredApplication{Descriptor: definition, EntryAuthorized: entryAuthorized, SettingsAuthorized: settingsAuthorized, UpdateAvailable: registered.UpdateAvailable, AvailableVersion: registered.AvailableVersion, UpdateCheckedAt: registered.UpdateCheckedAt, UpdateCheckError: registered.UpdateCheckError})
		}
	}
	return result, nil
}

func (s *Service) External() *ExternalStore { return s.external }

func (s *Service) AuthorizeExternalEntry(ctx context.Context, userID, workspaceID uuid.UUID, app ExternalApplication) (bool, error) {
	permissions := make(map[string]permission.Definition)
	for _, definition := range s.reader.Snapshot().Permissions {
		permissions[definition.Name] = definition
	}
	return s.authorize(ctx, userID, workspaceID, permissions, s.external.Descriptor(app).Access.Entry)
}

func (s *Service) OwnsExternalPermission(app ExternalApplication, name string) bool {
	for _, item := range app.Permissions {
		if item.Name == name {
			return true
		}
	}
	return false
}

// EvaluatePermission is the narrow authorization decision exposed to trusted
// services. It deliberately returns only a boolean and resolves scope from the
// central permission catalog rather than trusting a caller-provided scope.
func (s *Service) EvaluatePermission(ctx context.Context, userID, workspaceID uuid.UUID, name string) (bool, error) {
	definition, ok := s.reader.SnapshotPermission(name)
	if !ok {
		return false, fmt.Errorf("permission %q is not registered", name)
	}
	if definition.Scope == permission.ScopePlatform {
		return s.authorizer.Can(ctx, userID, name)
	}
	if definition.Scope == permission.ScopeWorkspace {
		return s.authorizer.CanInWorkspace(ctx, userID, workspaceID, name)
	}
	return false, fmt.Errorf("permission %q has unsupported scope", name)
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

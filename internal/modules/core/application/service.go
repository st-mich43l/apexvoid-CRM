package application

import (
	"context"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	frameworkerrors "github.com/st-mich43l/apexvoid-CRM/internal/framework/errors"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/api"
)

type Service struct{ reader api.MetadataReader }

var _ api.FrameworkReader = (*Service)(nil)

func NewService(reader api.MetadataReader) *Service { return &Service{reader: reader} }

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
		result = append(result, metadata.PermissionMetadata{Name: item.Name, Module: item.Module, DisplayName: item.DisplayName, Description: item.Description})
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

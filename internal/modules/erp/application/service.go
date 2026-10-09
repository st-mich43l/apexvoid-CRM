package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/domain"
)

type Repository interface {
	Create(context.Context, uuid.UUID, uuid.UUID, domain.Input) (domain.Product, error)
	List(context.Context, uuid.UUID, string, bool, int, int) ([]domain.Product, int, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Product, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int, domain.Input) (domain.Product, error)
	SetArchived(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int, bool) (domain.Product, error)
}

type Service struct { repository Repository }
func New(repository Repository) *Service { return &Service{repository: repository} }
func (s *Service) Create(ctx context.Context, workspaceID, actorID uuid.UUID, input domain.Input) (domain.Product, error) {
	normalized, err := input.Normalize()
	if err != nil { return domain.Product{}, err }
	return s.repository.Create(ctx, workspaceID, actorID, normalized)
}
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, search string, includeArchived bool, page, limit int) ([]domain.Product, int, error) {
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 25 }
	return s.repository.List(ctx, workspaceID, search, includeArchived, page, limit)
}
func (s *Service) Get(ctx context.Context, workspaceID, id uuid.UUID) (domain.Product, error) {
	return s.repository.Get(ctx, workspaceID, id)
}
func (s *Service) Update(ctx context.Context, workspaceID, actorID, id uuid.UUID, version int, input domain.Input) (domain.Product, error) {
	normalized, err := input.Normalize()
	if err != nil || version < 1 { return domain.Product{}, domain.ErrInvalid }
	return s.repository.Update(ctx, workspaceID, actorID, id, version, normalized)
}
func (s *Service) SetArchived(ctx context.Context, workspaceID, actorID, id uuid.UUID, version int, archived bool) (domain.Product, error) {
	if version < 1 { return domain.Product{}, domain.ErrInvalid }
	return s.repository.SetArchived(ctx, workspaceID, actorID, id, version, archived)
}

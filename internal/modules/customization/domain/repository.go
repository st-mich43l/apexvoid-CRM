package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	ListFields(context.Context, uuid.UUID, string, bool) ([]RuntimeField, error)
	CreateField(context.Context, *RuntimeField) error
	UpdateField(context.Context, *RuntimeField) error
	ListSections(context.Context, uuid.UUID, string) ([]FormSection, error)
	CreateSection(context.Context, *FormSection) error
	UpdateSection(context.Context, *FormSection) error
	ListViews(context.Context, uuid.UUID, uuid.UUID, string) ([]SavedView, error)
	CreateView(context.Context, *SavedView) error
	UpdateView(context.Context, *SavedView) error
}

package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/domain"
)

// ProductReader is the narrow shared catalog contract consumed by business apps.
// Consumers must not directly query or mutate ERP catalog persistence.
type ProductReader interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Product,error)
}

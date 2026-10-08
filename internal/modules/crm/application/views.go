package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	customdomain "github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/domain"
)

// resolveView loads only views visible to the requesting user in the verified
// workspace. Never accept a client-supplied saved view definition as authority.
func (s *Service) resolveView(ctx context.Context, workspaceID, userID, viewID uuid.UUID, entity string) ([]domain.ViewCondition, string, bool, bool, string, error) {
	if s.custom == nil {
		return nil, "", false, false, "", fmt.Errorf("customization is unavailable")
	}
	views, err := s.custom.ListViews(ctx, workspaceID, userID, entity)
	if err != nil {
		return nil, "", false, false, "", err
	}
	for _, view := range views {
		if view.ID != viewID {
			continue
		}
		schema, err := s.custom.EffectiveSchema(ctx, workspaceID, entity)
		if err != nil {
			return nil, "", false, false, "", err
		}
		fields := make(map[string]customdomain.EffectiveField, len(schema.Fields))
		for _, field := range schema.Fields {
			fields[field.Key] = field
		}
		if err := view.Validate(fields); err != nil {
			return nil, "", false, false, "", err
		}
		conditions := make([]domain.ViewCondition, 0, len(view.Filters))
		for _, filter := range view.Filters {
			field := fields[filter.Field]
			if field.Source == customdomain.FieldSourceCustom && !field.Visible {
				return nil, "", false, false, "", fmt.Errorf("saved view references a hidden field")
			}
			conditions = append(conditions, domain.ViewCondition{Field: filter.Field, Operator: filter.Operator, Value: filter.Value, Custom: field.Source == customdomain.FieldSourceCustom, Type: string(field.Type)})
		}
		sortCustom, sortType := false, ""
		if view.SortField != "" {
			field := fields[view.SortField]
			sortCustom, sortType = field.Source == customdomain.FieldSourceCustom, string(field.Type)
			if sortCustom && !field.Visible {
				return nil, "", false, false, "", fmt.Errorf("saved view sort field is hidden")
			}
		}
		return conditions, view.SortField, view.SortDirection == "desc", sortCustom, sortType, nil
	}
	return nil, "", false, false, "", domain.ErrNotFound
}

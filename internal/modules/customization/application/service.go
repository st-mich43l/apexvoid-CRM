package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	contactsapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Service struct {
	repository   domain.Repository
	metadata     *metadata.Registry
	contacts     contactsapi.CustomFieldReader
	transactions *database.TxManager
}

type Dependencies struct {
	Repository   domain.Repository
	Metadata     *metadata.Registry
	Contacts     contactsapi.CustomFieldReader
	Transactions *database.TxManager
}

func NewService(d Dependencies) *Service {
	return &Service{repository: d.Repository, metadata: d.Metadata, contacts: d.Contacts, transactions: d.Transactions}
}

func (s *Service) EffectiveSchema(ctx context.Context, workspaceID uuid.UUID, entityName string) (domain.EffectiveSchema, error) {
	definition, err := s.workspaceEntity(entityName)
	if err != nil {
		return domain.EffectiveSchema{}, err
	}
	sections, err := s.repository.ListSections(ctx, workspaceID, entityName)
	if err != nil {
		return domain.EffectiveSchema{}, err
	}
	fields := make([]domain.EffectiveField, 0, len(definition.Fields)+8)
	keys := map[string]struct{}{}
	for index, item := range definition.Fields {
		keys[item.Name] = struct{}{}
		fields = append(fields, domain.EffectiveField{Key: item.Name, Label: item.DisplayName, Type: item.Type, Description: item.Description, Required: item.Required, ReadOnly: item.ReadOnly, Source: domain.FieldSourceBuiltIn, Visible: true, DisplayOrder: index})
	}
	if entityName == "contacts.contact" && s.contacts != nil {
		legacy, listErr := s.contacts.ListCustomFields(ctx, workspaceID, true)
		if listErr != nil {
			return domain.EffectiveSchema{}, listErr
		}
		for _, item := range legacy {
			keys[item.Key] = struct{}{}
			fields = append(fields, domain.EffectiveField{Key: item.Key, Label: item.Label, Type: contactFieldType(string(item.Type)), Description: item.Description, Required: item.Required, Source: domain.FieldSourceCustom, Options: append([]string{}, item.Options...), Visible: item.Active, DisplayOrder: item.DisplayOrder})
		}
	}
	runtime, err := s.repository.ListFields(ctx, workspaceID, entityName, true)
	if err != nil {
		return domain.EffectiveSchema{}, err
	}
	for _, item := range runtime {
		if _, exists := keys[item.Key]; exists {
			return domain.EffectiveSchema{}, fmt.Errorf("runtime field %q conflicts with compiled metadata", item.Key)
		}
		keys[item.Key] = struct{}{}
		fields = append(fields, toEffective(item))
	}
	domain.SortFields(fields)
	return domain.EffectiveSchema{Entity: entityName, Fields: fields, Sections: sections}, nil
}

// contactFieldType translates Phase 5 Contacts types into the shared field vocabulary.
func contactFieldType(kind string) field.Type {
	switch kind {
	case "text":
		return field.String
	case "number":
		return field.Decimal
	case "boolean":
		return field.Boolean
	case "date":
		return field.Date
	case "selection":
		return field.Enum
	default:
		return field.Type(kind)
	}
}

func (s *Service) CreateField(ctx context.Context, item domain.RuntimeField) (domain.RuntimeField, error) {
	if _, err := s.workspaceEntity(item.Entity); err != nil {
		return domain.RuntimeField{}, err
	}
	if item.Entity == "contacts.contact" {
		return domain.RuntimeField{}, fmt.Errorf("contacts fields are managed through the established contacts field API")
	}
	item.ID = uuid.New()
	item.Key = strings.TrimSpace(item.Key)
	item.Label = strings.TrimSpace(item.Label)
	item.Description = strings.TrimSpace(item.Description)
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	if err := item.Validate(); err != nil {
		return domain.RuntimeField{}, err
	}
	schema, err := s.EffectiveSchema(ctx, item.WorkspaceID, item.Entity)
	if err != nil {
		return domain.RuntimeField{}, err
	}
	for _, existing := range schema.Fields {
		if existing.Key == item.Key {
			return domain.RuntimeField{}, fmt.Errorf("field key is already in use")
		}
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateField(tx, &item) }); err != nil {
		return domain.RuntimeField{}, err
	}
	return item, nil
}

// ListFields returns the managed inventory, including inactive definitions.
// EffectiveSchema remains a consumer schema and intentionally hides inactive fields.
func (s *Service) ListFields(ctx context.Context, workspaceID uuid.UUID, entityName string) ([]domain.RuntimeField, error) {
	if _, err := s.workspaceEntity(entityName); err != nil {
		return nil, err
	}
	if entityName == "contacts.contact" {
		return nil, fmt.Errorf("contacts fields are managed through the existing contacts field API")
	}
	return s.repository.ListFields(ctx, workspaceID, entityName, false)
}

// PatchField changes only supplied properties; key, type, entity and workspace are immutable.
func (s *Service) PatchField(ctx context.Context, workspaceID, id uuid.UUID, patch domain.FieldPatch) (domain.RuntimeField, error) {
	var updated domain.RuntimeField
	err := s.withTransaction(ctx, func(tx context.Context) error {
		current, err := s.repository.GetField(tx, workspaceID, id)
		if err != nil {
			return err
		}
		if patch.Label != nil {
			current.Label = strings.TrimSpace(*patch.Label)
		}
		if patch.Description != nil {
			current.Description = strings.TrimSpace(*patch.Description)
		}
		if patch.Required != nil {
			current.Required = *patch.Required
		}
		if patch.SetDefault {
			current.DefaultValue = patch.DefaultValue
		}
		if patch.Options != nil {
			current.Options = *patch.Options
		}
		if patch.Visible != nil {
			current.Visible = *patch.Visible
		}
		if patch.DisplayOrder != nil {
			current.DisplayOrder = *patch.DisplayOrder
		}
		if patch.SetSection {
			current.SectionID = patch.SectionID
		}
		if patch.Active != nil {
			current.Active = *patch.Active
		}
		if err := current.Validate(); err != nil {
			return err
		}
		current.UpdatedAt = time.Now().UTC()
		if err := s.repository.UpdateField(tx, &current); err != nil {
			return err
		}
		updated = current
		return nil
	})
	return updated, err
}

func (s *Service) CreateSection(ctx context.Context, item domain.FormSection) (domain.FormSection, error) {
	if _, err := s.workspaceEntity(item.Entity); err != nil {
		return domain.FormSection{}, err
	}
	item.ID = uuid.New()
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	if err := item.Validate(); err != nil {
		return domain.FormSection{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateSection(tx, &item) }); err != nil {
		return domain.FormSection{}, err
	}
	return item, nil
}

func (s *Service) ListSections(ctx context.Context, workspaceID uuid.UUID, entityName string) ([]domain.FormSection, error) {
	if _, err := s.workspaceEntity(entityName); err != nil {
		return nil, err
	}
	return s.repository.ListSections(ctx, workspaceID, entityName)
}

func (s *Service) UpdateSection(ctx context.Context, item domain.FormSection) (domain.FormSection, error) {
	if _, err := s.workspaceEntity(item.Entity); err != nil {
		return domain.FormSection{}, err
	}
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.UpdatedAt = time.Now().UTC()
	if err := item.Validate(); err != nil {
		return domain.FormSection{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateSection(tx, &item) }); err != nil {
		return domain.FormSection{}, err
	}
	return item, nil
}

func (s *Service) ListViews(ctx context.Context, workspaceID, userID uuid.UUID, entityName string) ([]domain.SavedView, error) {
	if _, err := s.workspaceEntity(entityName); err != nil {
		return nil, err
	}
	return s.repository.ListViews(ctx, workspaceID, userID, entityName)
}

func (s *Service) CreateView(ctx context.Context, item domain.SavedView) (domain.SavedView, error) {
	schema, err := s.EffectiveSchema(ctx, item.WorkspaceID, item.Entity)
	if err != nil {
		return domain.SavedView{}, err
	}
	item.ID = uuid.New()
	item.Name = strings.TrimSpace(item.Name)
	item.SortDirection = strings.ToLower(strings.TrimSpace(item.SortDirection))
	if item.SortDirection == "" {
		item.SortDirection = "asc"
	}
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	if err := item.Validate(fieldMetadata(schema)); err != nil {
		return domain.SavedView{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateView(tx, &item) }); err != nil {
		return domain.SavedView{}, err
	}
	return item, nil
}

func (s *Service) UpdateView(ctx context.Context, workspaceID, userID, id uuid.UUID, entityName string, input domain.SavedView) (domain.SavedView, error) {
	views, err := s.repository.ListViews(ctx, workspaceID, userID, entityName)
	if err != nil {
		return domain.SavedView{}, err
	}
	var current domain.SavedView
	for _, view := range views {
		if view.ID == id {
			current = view
			break
		}
	}
	if current.ID == uuid.Nil {
		return domain.SavedView{}, domain.ErrNotFound
	}
	schema, err := s.EffectiveSchema(ctx, workspaceID, entityName)
	if err != nil {
		return domain.SavedView{}, err
	}
	current.Name, current.Shared, current.Filters, current.Columns = strings.TrimSpace(input.Name), input.Shared, input.Filters, input.Columns
	current.SortField, current.SortDirection = input.SortField, strings.ToLower(strings.TrimSpace(input.SortDirection))
	if current.SortDirection == "" {
		current.SortDirection = "asc"
	}
	current.UpdatedAt = time.Now().UTC()
	if err := current.Validate(fieldMetadata(schema)); err != nil {
		return domain.SavedView{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateView(tx, &current) }); err != nil {
		return domain.SavedView{}, err
	}
	return current, nil
}

func (s *Service) DeleteView(ctx context.Context, workspaceID, userID, id uuid.UUID, entityName string) error {
	views, err := s.repository.ListViews(ctx, workspaceID, userID, entityName)
	if err != nil {
		return err
	}
	for _, view := range views {
		if view.ID == id {
			return s.withTransaction(ctx, func(tx context.Context) error {
				return s.repository.DeleteView(tx, workspaceID, entityName, id, view.OwnerUserID)
			})
		}
	}
	return domain.ErrNotFound
}

func (s *Service) ValidateCustomValues(ctx context.Context, workspaceID uuid.UUID, entityName string, values map[string]any) error {
	schema, err := s.EffectiveSchema(ctx, workspaceID, entityName)
	if err != nil {
		return err
	}
	fields := map[string]domain.EffectiveField{}
	for _, item := range schema.Fields {
		if item.Source == domain.FieldSourceCustom && item.Visible {
			fields[item.Key] = item
			if item.Required {
				if _, ok := values[item.Key]; !ok {
					return fmt.Errorf("custom field %q is required", item.Key)
				}
			}
		}
	}
	for key, value := range values {
		item, ok := fields[key]
		if !ok {
			return fmt.Errorf("unknown custom field %q", key)
		}
		if !domain.ValidValue(item.Type, item.Options, value) {
			return fmt.Errorf("custom field %q has an invalid value", key)
		}
	}
	return nil
}

func (s *Service) workspaceEntity(name string) (entity.Definition, error) {
	if s.metadata == nil {
		return entity.Definition{}, fmt.Errorf("metadata registry is not configured")
	}
	definition, ok := s.metadata.Entity(name)
	if !ok || definition.Scope != entity.ScopeWorkspace {
		return entity.Definition{}, fmt.Errorf("workspace entity was not found")
	}
	return definition, nil
}

func toEffective(item domain.RuntimeField) domain.EffectiveField {
	return domain.EffectiveField{Key: item.Key, Label: item.Label, Type: item.Type, Description: item.Description, Required: item.Required, Source: domain.FieldSourceCustom, DefaultValue: item.DefaultValue, Options: append([]string{}, item.Options...), Visible: item.Visible && item.Active, DisplayOrder: item.DisplayOrder, SectionID: item.SectionID}
}

func fieldMetadata(schema domain.EffectiveSchema) map[string]domain.EffectiveField {
	result := make(map[string]domain.EffectiveField, len(schema.Fields))
	for _, item := range schema.Fields {
		result[item.Key] = item
	}
	return result
}

func (s *Service) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if s.transactions == nil {
		return fn(ctx)
	}
	return s.transactions.WithTransaction(ctx, fn)
}

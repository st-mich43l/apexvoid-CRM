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
			fields = append(fields, domain.EffectiveField{Key: item.Key, Label: item.Label, Type: field.Type(item.Type), Description: item.Description, Required: item.Required, Source: domain.FieldSourceCustom, Options: append([]string{}, item.Options...), Visible: item.Active, DisplayOrder: item.DisplayOrder})
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
	item.Active = true
	item.Visible = true
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

func (s *Service) UpdateField(ctx context.Context, item domain.RuntimeField) (domain.RuntimeField, error) {
	if _, err := s.workspaceEntity(item.Entity); err != nil {
		return domain.RuntimeField{}, err
	}
	if item.Entity == "contacts.contact" {
		return domain.RuntimeField{}, fmt.Errorf("contacts fields are managed through the established contacts field API")
	}
	item.Key = strings.TrimSpace(item.Key)
	item.Label = strings.TrimSpace(item.Label)
	item.Description = strings.TrimSpace(item.Description)
	item.UpdatedAt = time.Now().UTC()
	if err := item.Validate(); err != nil {
		return domain.RuntimeField{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateField(tx, &item) }); err != nil {
		return domain.RuntimeField{}, err
	}
	return item, nil
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
	if err := item.Validate(fieldNames(schema)); err != nil {
		return domain.SavedView{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateView(tx, &item) }); err != nil {
		return domain.SavedView{}, err
	}
	return item, nil
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

func fieldNames(schema domain.EffectiveSchema) map[string]struct{} {
	result := make(map[string]struct{}, len(schema.Fields))
	for _, item := range schema.Fields {
		result[item.Key] = struct{}{}
	}
	return result
}

func (s *Service) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if s.transactions == nil {
		return fn(ctx)
	}
	return s.transactions.WithTransaction(ctx, fn)
}

package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	contactsapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Service struct {
	repository     domain.Repository
	transactions   *database.TxManager
	access         organizationapi.WorkspaceAccess
	events         *event.Bus
	logger         *slog.Logger
	store          domain.FileStore
	maxUploadBytes int64
}

type Dependencies struct {
	Repository     domain.Repository
	Transactions   *database.TxManager
	Access         organizationapi.WorkspaceAccess
	Events         *event.Bus
	Logger         *slog.Logger
	Store          domain.FileStore
	MaxUploadBytes int64
}

func NewService(d Dependencies) *Service {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.MaxUploadBytes <= 0 {
		d.MaxUploadBytes = 25 * 1024 * 1024
	}
	return &Service{repository: d.Repository, transactions: d.Transactions, access: d.Access, events: d.Events, logger: d.Logger, store: d.Store, maxUploadBytes: d.MaxUploadBytes}
}

func (s *Service) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (domain.ContactSummary, error) {
	c, err := s.repository.GetContact(ctx, workspaceID, id)
	if err != nil {
		return domain.ContactSummary{}, err
	}
	return domain.ContactSummary{ID: c.ID, WorkspaceID: c.WorkspaceID, Kind: c.Kind, DisplayName: c.DisplayName, Email: c.Email}, nil
}

func (s *Service) GetContact(ctx context.Context, w, id uuid.UUID) (domain.Contact, error) {
	c, err := s.repository.GetContact(ctx, w, id)
	if err != nil {
		return domain.Contact{}, err
	}
	return *c, nil
}
func (s *Service) ListContacts(ctx context.Context, w uuid.UUID, filter domain.ListFilter) ([]domain.Contact, int, error) {
	return s.repository.ListContacts(ctx, w, filter)
}

func (s *Service) CreateContact(ctx context.Context, c domain.Contact) (domain.Contact, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.Status == "" {
		c.Status = domain.StatusActive
	}
	c.DisplayName = strings.TrimSpace(c.DisplayName)
	c.Email = strings.TrimSpace(c.Email)
	c.Phone = strings.TrimSpace(c.Phone)
	c.Website = strings.TrimSpace(c.Website)
	if err := s.validateCustomValues(ctx, c.WorkspaceID, c.CustomValues); err != nil {
		return domain.Contact{}, err
	}
	if err := c.Validate(); err != nil {
		return domain.Contact{}, err
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateContact(tx, &c) }); err != nil {
		return domain.Contact{}, err
	}
	s.publish(ctx, "contacts.contact.created", domain.ContactCreated{ContactID: c.ID, WorkspaceID: c.WorkspaceID})
	return c, nil
}

func (s *Service) UpdateContact(ctx context.Context, c domain.Contact) (domain.Contact, error) {
	current, err := s.repository.GetContact(ctx, c.WorkspaceID, c.ID)
	if err != nil {
		return domain.Contact{}, err
	}
	if c.Status == "" {
		c.Status = current.Status
	}
	c.DisplayName = strings.TrimSpace(c.DisplayName)
	c.Email = strings.TrimSpace(c.Email)
	c.Phone = strings.TrimSpace(c.Phone)
	c.Website = strings.TrimSpace(c.Website)
	if err := s.validateCustomValues(ctx, c.WorkspaceID, c.CustomValues); err != nil {
		return domain.Contact{}, err
	}
	if err := c.Validate(); err != nil {
		return domain.Contact{}, err
	}
	c.CreatedAt, c.CreatedBy = current.CreatedAt, current.CreatedBy
	c.UpdatedAt = time.Now().UTC()
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateContact(tx, &c) }); err != nil {
		return domain.Contact{}, err
	}
	s.publish(ctx, "contacts.contact.updated", domain.ContactUpdated{ContactID: c.ID, WorkspaceID: c.WorkspaceID})
	return c, nil
}

func (s *Service) SetContactStatus(ctx context.Context, w, id uuid.UUID, status domain.ContactStatus, by uuid.UUID) error {
	if status != domain.StatusActive && status != domain.StatusArchived {
		return domain.ErrInvalidStatus
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.SetContactStatus(tx, w, id, status, by) }); err != nil {
		return err
	}
	if status == domain.StatusArchived {
		s.publish(ctx, "contacts.contact.archived", domain.ContactArchived{ContactID: id, WorkspaceID: w})
	} else {
		s.publish(ctx, "contacts.contact.updated", domain.ContactUpdated{ContactID: id, WorkspaceID: w})
	}
	return nil
}

func (s *Service) ListRelationships(ctx context.Context, w, id uuid.UUID) ([]domain.Relationship, error) {
	if err := s.requireContact(ctx, w, id); err != nil {
		return nil, err
	}
	return s.repository.ListRelationships(ctx, w, id)
}
func (s *Service) ReplaceRelationships(ctx context.Context, w, id uuid.UUID, items []domain.Relationship) error {
	if err := s.requireContact(ctx, w, id); err != nil {
		return err
	}
	for i := range items {
		if err := items[i].Validate(); err != nil {
			return err
		}
		if items[i].ID == uuid.Nil {
			items[i].ID = uuid.New()
		}
		items[i].WorkspaceID = w
		person, err := s.repository.GetContact(ctx, w, items[i].PersonID)
		if err != nil {
			return err
		}
		company, err := s.repository.GetContact(ctx, w, items[i].CompanyID)
		if err != nil {
			return err
		}
		if person.Kind != domain.KindPerson || company.Kind != domain.KindCompany {
			return errors.New("relationships require a person and a company")
		}
		if items[i].PersonID != id && items[i].CompanyID != id {
			return errors.New("relationship must include the selected contact")
		}
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.ReplaceRelationships(tx, w, id, items) }); err != nil {
		return err
	}
	s.publish(ctx, "contacts.relationship.changed", domain.RelationshipChanged{ContactID: id, WorkspaceID: w})
	return nil
}

func (s *Service) ListTags(ctx context.Context, w uuid.UUID, active bool) ([]domain.Tag, error) {
	return s.repository.ListTags(ctx, w, active)
}
func (s *Service) CreateTag(ctx context.Context, t domain.Tag) (domain.Tag, error) {
	t.ID = uuid.New()
	t.Active = true
	t.Name = strings.TrimSpace(t.Name)
	t.Color = strings.TrimSpace(t.Color)
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	if err := t.Validate(); err != nil {
		return domain.Tag{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateTag(tx, &t) }); err != nil {
		return domain.Tag{}, err
	}
	return t, nil
}
func (s *Service) UpdateTag(ctx context.Context, t domain.Tag) (domain.Tag, error) {
	t.Name = strings.TrimSpace(t.Name)
	t.Color = strings.TrimSpace(t.Color)
	t.UpdatedAt = time.Now().UTC()
	if err := t.Validate(); err != nil {
		return domain.Tag{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateTag(tx, &t) }); err != nil {
		return domain.Tag{}, err
	}
	return t, nil
}
func (s *Service) ReplaceContactTags(ctx context.Context, w, c uuid.UUID, ids []uuid.UUID) error {
	if err := s.requireContact(ctx, w, c); err != nil {
		return err
	}
	tags, err := s.repository.ListTags(ctx, w, true)
	if err != nil {
		return err
	}
	allowed := map[uuid.UUID]bool{}
	for _, t := range tags {
		allowed[t.ID] = true
	}
	for _, id := range ids {
		if !allowed[id] {
			return fmt.Errorf("tag is not active in this workspace")
		}
	}
	return s.withTransaction(ctx, func(tx context.Context) error { return s.repository.ReplaceContactTags(tx, w, c, ids) })
}
func (s *Service) ContactTagIDs(ctx context.Context, w, c uuid.UUID) ([]uuid.UUID, error) {
	if err := s.requireContact(ctx, w, c); err != nil {
		return nil, err
	}
	return s.repository.ContactTagIDs(ctx, w, c)
}

func (s *Service) ListNotes(ctx context.Context, w, c uuid.UUID) ([]domain.Note, error) {
	if err := s.requireContact(ctx, w, c); err != nil {
		return nil, err
	}
	return s.repository.ListNotes(ctx, w, c)
}
func (s *Service) CreateNote(ctx context.Context, n domain.Note) (domain.Note, error) {
	if err := s.requireContact(ctx, n.WorkspaceID, n.ContactID); err != nil {
		return domain.Note{}, err
	}
	if err := n.Validate(); err != nil {
		return domain.Note{}, err
	}
	n.ID = uuid.New()
	n.CreatedAt = time.Now().UTC()
	n.UpdatedAt = n.CreatedAt
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateNote(tx, &n) }); err != nil {
		return domain.Note{}, err
	}
	return n, nil
}
func (s *Service) UpdateNote(ctx context.Context, n domain.Note) (domain.Note, error) {
	if err := n.Validate(); err != nil {
		return domain.Note{}, err
	}
	n.UpdatedAt = time.Now().UTC()
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateNote(tx, &n) }); err != nil {
		return domain.Note{}, err
	}
	return n, nil
}

func (s *Service) ListActivities(ctx context.Context, w uuid.UUID, c, user *uuid.UUID, status domain.ActivityStatus) ([]domain.Activity, error) {
	if c != nil {
		if err := s.requireContact(ctx, w, *c); err != nil {
			return nil, err
		}
	}
	return s.repository.ListActivities(ctx, w, c, user, status)
}
func (s *Service) ListActivitiesPage(ctx context.Context, w uuid.UUID, filter domain.ActivityListFilter) ([]domain.Activity, int, error) {
	if filter.ContactID != nil {
		if err := s.requireContact(ctx, w, *filter.ContactID); err != nil {
			return nil, 0, err
		}
	}
	return s.repository.ListActivitiesPage(ctx, w, filter)
}
func (s *Service) FindActivity(ctx context.Context, w, id uuid.UUID) (domain.Activity, error) {
	item, err := s.repository.GetActivityByID(ctx, w, id)
	if err != nil {
		return domain.Activity{}, err
	}
	return *item, nil
}
func (s *Service) CreateActivity(ctx context.Context, a domain.Activity) (domain.Activity, error) {
	if err := s.validateActivity(ctx, &a); err != nil {
		return domain.Activity{}, err
	}
	a.ID = uuid.New()
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateActivity(tx, &a) }); err != nil {
		return domain.Activity{}, err
	}
	s.publish(ctx, "contacts.activity.created", domain.ActivityCreated{ActivityID: a.ID, WorkspaceID: a.WorkspaceID})
	return a, nil
}
func (s *Service) UpdateActivity(ctx context.Context, a domain.Activity) (domain.Activity, error) {
	current, err := s.repository.GetActivityByID(ctx, a.WorkspaceID, a.ID)
	if err != nil {
		return domain.Activity{}, err
	}
	if a.Status == "" {
		a.Status = current.Status
	}
	if a.Status != current.Status {
		transition := *current
		if err := transition.TransitionTo(a.Status); err != nil {
			return domain.Activity{}, err
		}
		if a.CompletedAt == nil {
			a.CompletedAt = transition.CompletedAt
		}
	}
	if err := s.validateActivity(ctx, &a); err != nil {
		return domain.Activity{}, err
	}
	a.UpdatedAt = time.Now().UTC()
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateActivity(tx, &a) }); err != nil {
		return domain.Activity{}, err
	}
	if a.Status == domain.ActivityCompleted {
		s.publish(ctx, "contacts.activity.completed", domain.ActivityCompletedEvent{ActivityID: a.ID, WorkspaceID: a.WorkspaceID})
	}
	return a, nil
}

func (s *Service) ListAttachments(ctx context.Context, w, c uuid.UUID) ([]domain.Attachment, error) {
	if err := s.requireContact(ctx, w, c); err != nil {
		return nil, err
	}
	return s.repository.ListAttachments(ctx, w, c)
}
func (s *Service) UploadAttachment(ctx context.Context, a domain.Attachment, reader io.Reader, contentType string) (domain.Attachment, error) {
	if s.store == nil {
		return domain.Attachment{}, errors.New("attachment storage is not configured")
	}
	if err := s.requireContact(ctx, a.WorkspaceID, a.ContactID); err != nil {
		return domain.Attachment{}, err
	}
	a.ID = uuid.New()
	a.FileName = safeFileName(a.FileName)
	a.ContentType = contentType
	a.CreatedAt = time.Now().UTC()
	key, size, err := s.store.Save(ctx, a.WorkspaceID.String()+"/"+a.ContactID.String(), reader, s.maxUploadBytes)
	if err != nil {
		return domain.Attachment{}, err
	}
	a.Size = size
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateAttachment(tx, &a, key) }); err != nil {
		_ = s.store.Delete(ctx, key)
		return domain.Attachment{}, err
	}
	return a, nil
}
func (s *Service) MaxUploadBytes() int64 { return s.maxUploadBytes }
func (s *Service) OpenAttachment(ctx context.Context, w, c, id uuid.UUID) (domain.Attachment, io.ReadCloser, error) {
	a, key, err := s.repository.GetAttachment(ctx, w, c, id)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	reader, err := s.store.Open(ctx, key)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	return a, reader, nil
}
func (s *Service) DeleteAttachment(ctx context.Context, w, c, id uuid.UUID) error {
	a, key, err := s.repository.GetAttachment(ctx, w, c, id)
	if err != nil {
		return err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.DeleteAttachment(tx, w, c, id) }); err != nil {
		return err
	}
	if err := s.store.Delete(ctx, key); err != nil {
		s.logger.Error("failed to remove attachment object", "attachment_id", a.ID, "error", err)
	}
	return nil
}

func (s *Service) ListCustomFields(ctx context.Context, w uuid.UUID, active bool) ([]domain.CustomFieldDefinition, error) {
	return s.repository.ListCustomFields(ctx, w, active)
}
func (s *Service) CreateCustomField(ctx context.Context, f domain.CustomFieldDefinition) (domain.CustomFieldDefinition, error) {
	f.ID = uuid.New()
	f.Active = true
	f.CreatedAt = time.Now().UTC()
	f.UpdatedAt = f.CreatedAt
	if err := f.Validate(); err != nil {
		return domain.CustomFieldDefinition{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.CreateCustomField(tx, &f) }); err != nil {
		return domain.CustomFieldDefinition{}, err
	}
	return f, nil
}
func (s *Service) UpdateCustomField(ctx context.Context, f domain.CustomFieldDefinition) (domain.CustomFieldDefinition, error) {
	fields, err := s.repository.ListCustomFields(ctx, f.WorkspaceID, false)
	if err != nil {
		return domain.CustomFieldDefinition{}, err
	}
	var current *domain.CustomFieldDefinition
	for i := range fields {
		if fields[i].ID == f.ID {
			current = &fields[i]
			break
		}
	}
	if current == nil {
		return domain.CustomFieldDefinition{}, domain.ErrNotFound
	}
	if f.Key != current.Key {
		return domain.CustomFieldDefinition{}, errors.New("custom field key cannot be changed")
	}
	if f.Type != current.Type {
		count, err := s.repository.CountContactsWithCustomFieldValues(ctx, f.WorkspaceID, f.Key)
		if err != nil {
			return domain.CustomFieldDefinition{}, err
		}
		if count > 0 {
			return domain.CustomFieldDefinition{}, errors.New("custom field type cannot change while values exist")
		}
	}
	if f.Type == domain.FieldSelection && current.Type == domain.FieldSelection {
		removed, err := s.repository.CountContactsUsingCustomFieldOptions(ctx, f.WorkspaceID, f.Key, f.Options)
		if err != nil {
			return domain.CustomFieldDefinition{}, err
		}
		if removed > 0 {
			return domain.CustomFieldDefinition{}, errors.New("custom field options cannot remove values used by existing contacts")
		}
	}
	if f.Required && !current.Required {
		missing, err := s.repository.CountContactsMissingCustomField(ctx, f.WorkspaceID, f.Key)
		if err != nil {
			return domain.CustomFieldDefinition{}, err
		}
		if missing > 0 {
			return domain.CustomFieldDefinition{}, errors.New("custom field cannot be required while existing contacts are missing a value")
		}
	}
	f.UpdatedAt = time.Now().UTC()
	if err := f.Validate(); err != nil {
		return domain.CustomFieldDefinition{}, err
	}
	if err := s.withTransaction(ctx, func(tx context.Context) error { return s.repository.UpdateCustomField(tx, &f) }); err != nil {
		return domain.CustomFieldDefinition{}, err
	}
	return f, nil
}

func (s *Service) requireContact(ctx context.Context, w, id uuid.UUID) error {
	_, err := s.repository.GetContact(ctx, w, id)
	return err
}
func (s *Service) validateActivity(ctx context.Context, a *domain.Activity) error {
	if a.Status == "" {
		a.Status = domain.ActivityPlanned
	}
	if a.AssignedUserID == uuid.Nil {
		return errors.New("assigned user is required")
	}
	if err := a.Validate(); err != nil {
		return err
	}
	if !mustBeActive(ctx, s.repository, a.WorkspaceID, a.AssignedUserID) {
		return domain.ErrInactiveMember
	}
	if a.RelatedContactID != nil {
		return s.requireContact(ctx, a.WorkspaceID, *a.RelatedContactID)
	}
	return nil
}
func mustBeActive(ctx context.Context, repository domain.Repository, w, user uuid.UUID) bool {
	ok, err := repository.IsActiveWorkspaceMember(ctx, w, user)
	return err == nil && ok
}
func (s *Service) validateCustomValues(ctx context.Context, w uuid.UUID, values map[string]any) error {
	if len(values) == 0 {
		fields, err := s.repository.ListCustomFields(ctx, w, true)
		if err != nil {
			return err
		}
		for _, f := range fields {
			if f.Required {
				return fmt.Errorf("custom field %q is required", f.Key)
			}
		}
		return nil
	}
	fields, err := s.repository.ListCustomFields(ctx, w, true)
	if err != nil {
		return err
	}
	definitions := map[string]domain.CustomFieldDefinition{}
	for _, f := range fields {
		definitions[f.Key] = f
		if f.Required {
			if _, ok := values[f.Key]; !ok {
				return fmt.Errorf("custom field %q is required", f.Key)
			}
		}
	}
	for key, value := range values {
		f, ok := definitions[key]
		if !ok {
			return fmt.Errorf("unknown custom field %q", key)
		}
		if !validCustomValue(f, value) {
			return fmt.Errorf("%w: %s", domain.ErrInvalidCustomValue, key)
		}
	}
	return nil
}
func validCustomValue(f domain.CustomFieldDefinition, value any) bool {
	switch f.Type {
	case domain.FieldText:
		_, ok := value.(string)
		return ok
	case domain.FieldNumber:
		_, ok := value.(float64)
		return ok
	case domain.FieldBoolean:
		_, ok := value.(bool)
		return ok
	case domain.FieldDate:
		v, ok := value.(string)
		if !ok {
			return false
		}
		_, err := time.Parse("2006-01-02", v)
		return err == nil
	case domain.FieldSelection:
		v, ok := value.(string)
		if !ok {
			return false
		}
		for _, option := range f.Options {
			if v == option {
				return true
			}
		}
		return false
	}
	return false
}
func safeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		if r < 32 || r == '/' || r == '\\' {
			continue
		}
		b.WriteRune(r)
	}
	name = strings.TrimSpace(b.String())
	if name == "" || name == "." {
		return "attachment"
	}
	return name
}
func (s *Service) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if s.transactions == nil {
		return fn(ctx)
	}
	return s.transactions.WithTransaction(ctx, fn)
}
func (s *Service) publish(ctx context.Context, name string, payload any) {
	if s.events == nil { return }
	database.AfterCommit(ctx, func(ctx context.Context) {
		var err error
		switch value := payload.(type) {
		case domain.ContactCreated:
			err = event.Publish(s.events, ctx, name, value)
		case domain.ContactUpdated:
			err = event.Publish(s.events, ctx, name, value)
		case domain.ContactArchived:
			err = event.Publish(s.events, ctx, name, value)
		case domain.RelationshipChanged:
			err = event.Publish(s.events, ctx, name, value)
		case domain.ActivityCreated:
			err = event.Publish(s.events, ctx, name, value)
		case domain.ActivityCompletedEvent:
			err = event.Publish(s.events, ctx, name, value)
		default:
			return
		}
		if err != nil {
			s.logger.Error("post-commit event publication failed", "event", name, "error", err)
		}
	})
}

var _ domain.ContactReader = (*Service)(nil)
var _ contactsapi.ContactReader = (*Service)(nil)

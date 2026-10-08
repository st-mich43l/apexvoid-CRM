package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Handler struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
	access        organizationapi.WorkspaceAccess
}

func NewHandler(service *application.Service, auth usersapi.Authenticator, workspace organizationapi.WorkspaceResolver, access organizationapi.WorkspaceAccess) *Handler {
	return &Handler{service: service, authenticator: auth, workspace: workspace, access: access}
}

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	current := routes.With(usersapi.RequireAuthentication(h.authenticator), organizationapi.RequireWorkspace(h.workspace))
	read := current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.contact.read"))
	read.Get("/contacts", h.listContacts)
	read.Get("/contacts/{id}", h.getContact)
	read.Get("/contacts/{id}/relationships", h.listRelationships)
	read.Get("/contacts/{id}/tags", h.getTags)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.note.read")).Get("/contacts/{id}/notes", h.listNotes)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.activity.read")).Get("/contacts/{id}/activities", h.listActivities)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.attachment.read")).Get("/contacts/{id}/attachments", h.listAttachments)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.contact.create")).Post("/contacts", h.createContact)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.contact.update")).Patch("/contacts/{id}", h.updateContact)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.contact.archive")).Post("/contacts/{id}/archive", h.archiveContact)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.contact.archive")).Post("/contacts/{id}/restore", h.restoreContact)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.contact.update")).Put("/contacts/{id}/relationships", h.replaceRelationships)
	read.Get("/contacts/tags", h.listTags)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.tag.manage")).Post("/contacts/tags", h.createTag)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.tag.manage")).Patch("/contacts/tags/{tagID}", h.updateTag)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.tag.manage")).Put("/contacts/{id}/tags", h.replaceTags)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.note.manage")).Post("/contacts/{id}/notes", h.createNote)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.note.manage")).Patch("/contacts/{id}/notes/{noteID}", h.updateNote)
	activityRead := current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.activity.read"))
	activityRead.Get("/activities", h.listWorkspaceActivities)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.activity.manage")).Post("/activities", h.createActivity)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.activity.manage")).Post("/contacts/{id}/activities", h.createContactActivity)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.activity.manage")).Patch("/activities/{id}", h.updateActivity)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.activity.manage")).Post("/activities/{id}/complete", h.completeActivity)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.activity.manage")).Post("/activities/{id}/cancel", h.cancelActivity)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.attachment.read")).Get("/contacts/{id}/attachments/{attachmentID}", h.downloadAttachment)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.attachment.manage")).Post("/contacts/{id}/attachments", h.uploadAttachment)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.attachment.manage")).Delete("/contacts/{id}/attachments/{attachmentID}", h.deleteAttachment)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.contact.read")).Get("/contacts/fields", h.listFields)
	fieldRoutes := current.With(organizationapi.RequireWorkspacePermission(h.access, "contacts.field.manage"))
	fieldRoutes.Post("/contacts/fields", h.createField)
	fieldRoutes.Patch("/contacts/fields/{fieldID}", h.updateField)
	return nil
}

type contactRequest struct {
	Kind         domain.ContactKind `json:"kind"`
	DisplayName  string             `json:"display_name"`
	Email        string             `json:"email"`
	Phone        string             `json:"phone"`
	Website      string             `json:"website"`
	Description  string             `json:"description"`
	CustomValues map[string]any     `json:"custom_values"`
}
type tagRequest struct {
	Name   string `json:"name"`
	Color  string `json:"color"`
	Active *bool  `json:"active"`
}
type noteRequest struct {
	Content string `json:"content"`
}
type relationshipRequest struct {
	PersonID         uuid.UUID `json:"person_id"`
	CompanyID        uuid.UUID `json:"company_id"`
	RelationshipType string    `json:"relationship_type"`
	JobTitle         string    `json:"job_title"`
	IsPrimary        bool      `json:"is_primary"`
}
type activityRequest struct {
	Title            string                `json:"title"`
	Description      string                `json:"description"`
	ActivityType     domain.ActivityType   `json:"activity_type"`
	RelatedContactID *uuid.UUID            `json:"related_contact_id"`
	AssignedUserID   uuid.UUID             `json:"assigned_user_id"`
	DueAt            *time.Time            `json:"due_at"`
	Status           domain.ActivityStatus `json:"status"`
}
type fieldRequest struct {
	Key          string                 `json:"key"`
	Label        string                 `json:"label"`
	Type         domain.CustomFieldType `json:"type"`
	Description  string                 `json:"description"`
	Required     bool                   `json:"required"`
	Options      []string               `json:"options"`
	DisplayOrder int                    `json:"display_order"`
	Active       *bool                  `json:"active"`
}

func (h *Handler) listContacts(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	page, limit := intQuery(r, "page", 1), intQuery(r, "limit", 25)
	tag := uuidQuery(r, "tag_id")
	items, total, err := h.service.ListContacts(r.Context(), workspace.WorkspaceID, domain.ListFilter{Search: r.URL.Query().Get("search"), Kind: domain.ContactKind(r.URL.Query().Get("kind")), Status: domain.ContactStatus(r.URL.Query().Get("status")), TagID: tag, Page: page, Limit: limit, Sort: r.URL.Query().Get("sort"), Desc: r.URL.Query().Get("desc") == "true"})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page, "limit": limit, "total": total})
}
func (h *Handler) getContact(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	c, err := h.service.GetContact(r.Context(), workspace.WorkspaceID, id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (h *Handler) createContact(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	principal, _ := usersapi.PrincipalFromContext(r.Context())
	var input contactRequest
	if !decode(w, r, &input) {
		return
	}
	c, err := h.service.CreateContact(r.Context(), domain.Contact{WorkspaceID: workspace.WorkspaceID, Kind: input.Kind, DisplayName: input.DisplayName, Email: input.Email, Phone: input.Phone, Website: input.Website, Description: input.Description, CustomValues: input.CustomValues, CreatedBy: principal.UserID, UpdatedBy: principal.UserID})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}
func (h *Handler) updateContact(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	principal, _ := usersapi.PrincipalFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var input contactRequest
	if !decode(w, r, &input) {
		return
	}
	c, err := h.service.UpdateContact(r.Context(), domain.Contact{ID: id, WorkspaceID: workspace.WorkspaceID, Kind: input.Kind, DisplayName: input.DisplayName, Email: input.Email, Phone: input.Phone, Website: input.Website, Description: input.Description, CustomValues: input.CustomValues, UpdatedBy: principal.UserID})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (h *Handler) archiveContact(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, domain.StatusArchived)
}
func (h *Handler) restoreContact(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, domain.StatusActive)
}
func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status domain.ContactStatus) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	principal, _ := usersapi.PrincipalFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if err := h.service.SetContactStatus(r.Context(), workspace.WorkspaceID, id, status, principal.UserID); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) listRelationships(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	x, err := h.service.ListRelationships(r.Context(), workspace.WorkspaceID, id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) replaceRelationships(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var input []relationshipRequest
	if !decode(w, r, &input) {
		return
	}
	values := make([]domain.Relationship, len(input))
	for i, x := range input {
		values[i] = domain.Relationship{PersonID: x.PersonID, CompanyID: x.CompanyID, RelationshipType: x.RelationshipType, JobTitle: x.JobTitle, IsPrimary: x.IsPrimary}
	}
	if err := h.service.ReplaceRelationships(r.Context(), workspace.WorkspaceID, id, values); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) listTags(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	x, err := h.service.ListTags(r.Context(), workspace.WorkspaceID, r.URL.Query().Get("active") != "false")
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) createTag(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	var input tagRequest
	if !decode(w, r, &input) {
		return
	}
	x, err := h.service.CreateTag(r.Context(), domain.Tag{WorkspaceID: workspace.WorkspaceID, Name: input.Name, Color: input.Color})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, x)
}
func (h *Handler) updateTag(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "tagID")
	if !ok {
		return
	}
	var input tagRequest
	if !decode(w, r, &input) {
		return
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	x, err := h.service.UpdateTag(r.Context(), domain.Tag{ID: id, WorkspaceID: workspace.WorkspaceID, Name: input.Name, Color: input.Color, Active: active})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) getTags(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	x, err := h.service.ContactTagIDs(r.Context(), workspace.WorkspaceID, id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) replaceTags(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var ids []uuid.UUID
	if !decode(w, r, &ids) {
		return
	}
	if err := h.service.ReplaceContactTags(r.Context(), workspace.WorkspaceID, id, ids); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	x, err := h.service.ListNotes(r.Context(), workspace.WorkspaceID, id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) createNote(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	principal, _ := usersapi.PrincipalFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var input noteRequest
	if !decode(w, r, &input) {
		return
	}
	x, err := h.service.CreateNote(r.Context(), domain.Note{WorkspaceID: workspace.WorkspaceID, ContactID: id, AuthorUserID: principal.UserID, Content: input.Content})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, x)
}
func (h *Handler) updateNote(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	contactID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	noteID, ok := pathUUID(w, r, "noteID")
	if !ok {
		return
	}
	var input noteRequest
	if !decode(w, r, &input) {
		return
	}
	x, err := h.service.UpdateNote(r.Context(), domain.Note{ID: noteID, WorkspaceID: workspace.WorkspaceID, ContactID: contactID, Content: input.Content})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) listActivities(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	x, err := h.service.ListActivities(r.Context(), workspace.WorkspaceID, &id, nil, domain.ActivityStatus(r.URL.Query().Get("status")))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) listWorkspaceActivities(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	x, err := h.service.ListActivities(r.Context(), workspace.WorkspaceID, uuidQuery(r, "contact_id"), uuidQuery(r, "assigned_user_id"), domain.ActivityStatus(r.URL.Query().Get("status")))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) createActivity(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	var input activityRequest
	if !decode(w, r, &input) {
		return
	}
	x, err := h.service.CreateActivity(r.Context(), domain.Activity{WorkspaceID: workspace.WorkspaceID, Title: input.Title, Description: input.Description, ActivityType: input.ActivityType, RelatedContactID: input.RelatedContactID, AssignedUserID: input.AssignedUserID, DueAt: input.DueAt, Status: input.Status})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, x)
}
func (h *Handler) createContactActivity(w http.ResponseWriter, r *http.Request) {
	contactID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var input activityRequest
	if !decode(w, r, &input) {
		return
	}
	input.RelatedContactID = &contactID
	h.createActivityWithInput(w, r, input)
}
func (h *Handler) createActivityWithInput(w http.ResponseWriter, r *http.Request, input activityRequest) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	x, err := h.service.CreateActivity(r.Context(), domain.Activity{WorkspaceID: workspace.WorkspaceID, Title: input.Title, Description: input.Description, ActivityType: input.ActivityType, RelatedContactID: input.RelatedContactID, AssignedUserID: input.AssignedUserID, DueAt: input.DueAt, Status: input.Status})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, x)
}
func (h *Handler) updateActivity(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var input activityRequest
	if !decode(w, r, &input) {
		return
	}
	x, err := h.service.UpdateActivity(r.Context(), domain.Activity{ID: id, WorkspaceID: workspace.WorkspaceID, Title: input.Title, Description: input.Description, ActivityType: input.ActivityType, RelatedContactID: input.RelatedContactID, AssignedUserID: input.AssignedUserID, DueAt: input.DueAt, Status: input.Status})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) completeActivity(w http.ResponseWriter, r *http.Request) {
	h.setActivityStatus(w, r, domain.ActivityCompleted)
}
func (h *Handler) cancelActivity(w http.ResponseWriter, r *http.Request) {
	h.setActivityStatus(w, r, domain.ActivityCancelled)
}
func (h *Handler) setActivityStatus(w http.ResponseWriter, r *http.Request, status domain.ActivityStatus) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	activity, err := h.service.FindActivity(r.Context(), workspace.WorkspaceID, id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	activity.Status = status
	if status == domain.ActivityCompleted {
		now := time.Now().UTC()
		activity.CompletedAt = &now
	}
	updated, err := h.service.UpdateActivity(r.Context(), activity)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) listAttachments(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	x, err := h.service.ListAttachments(r.Context(), workspace.WorkspaceID, id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	principal, _ := usersapi.PrincipalFromContext(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeDomainError(w, r, err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	defer file.Close()
	contentType, reader, err := sniff(file)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	x, err := h.service.UploadAttachment(r.Context(), domain.Attachment{WorkspaceID: workspace.WorkspaceID, ContactID: id, FileName: header.Filename, UploadedBy: principal.UserID}, reader, contentType)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, x)
}
func (h *Handler) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	contactID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	attachmentID, ok := pathUUID(w, r, "attachmentID")
	if !ok {
		return
	}
	a, reader, err := h.service.OpenAttachment(r.Context(), workspace.WorkspaceID, contactID, attachmentID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", a.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(a.FileName, `"`, "")+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(a.Size, 10))
	_, _ = io.Copy(w, reader)
}
func (h *Handler) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	c, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	a, ok := pathUUID(w, r, "attachmentID")
	if !ok {
		return
	}
	if err := h.service.DeleteAttachment(r.Context(), workspace.WorkspaceID, c, a); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listFields(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	x, err := h.service.ListCustomFields(r.Context(), workspace.WorkspaceID, r.URL.Query().Get("active") != "false")
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (h *Handler) createField(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	var input fieldRequest
	if !decode(w, r, &input) {
		return
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	x, err := h.service.CreateCustomField(r.Context(), domain.CustomFieldDefinition{WorkspaceID: workspace.WorkspaceID, Entity: "contacts.contact", Key: input.Key, Label: input.Label, Type: input.Type, Description: input.Description, Required: input.Required, Options: input.Options, DisplayOrder: input.DisplayOrder, Active: active})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, x)
}
func (h *Handler) updateField(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := pathUUID(w, r, "fieldID")
	if !ok {
		return
	}
	var input fieldRequest
	if !decode(w, r, &input) {
		return
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	x, err := h.service.UpdateCustomField(r.Context(), domain.CustomFieldDefinition{ID: id, WorkspaceID: workspace.WorkspaceID, Entity: "contacts.contact", Key: input.Key, Label: input.Label, Type: input.Type, Description: input.Description, Required: input.Required, Options: input.Options, DisplayOrder: input.DisplayOrder, Active: active})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, x)
}

func sniff(file multipart.File) (string, io.Reader, error) {
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", nil, err
	}
	if n == 0 {
		return "application/octet-stream", bytes.NewReader(nil), nil
	}
	contentType := http.DetectContentType(head[:n])
	if !allowedContentType(contentType) {
		return "", nil, errors.New("unsupported attachment content type")
	}
	return contentType, io.MultiReader(bytes.NewReader(head[:n]), file), nil
}
func allowedContentType(value string) bool {
	switch value {
	case "application/pdf", "text/plain", "text/csv", "image/jpeg", "image/png", "image/gif", "image/webp", "application/zip", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return true
	default:
		return false
	}
}
func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid identifier")
		return uuid.Nil, false
	}
	return id, true
}
func uuidQuery(r *http.Request, name string) *uuid.UUID {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil
	}
	return &id
}
func intQuery(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred"
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "Contact resource not found"
	case errors.Is(err, domain.ErrDuplicate):
		status, code, message = http.StatusConflict, "CONFLICT", "Contact resource already exists"
	case errors.Is(err, domain.ErrInvalidStatus), errors.Is(err, domain.ErrInvalidCustomValue), errors.Is(err, domain.ErrInactiveMember):
		status, code, message = http.StatusBadRequest, "VALIDATION_ERROR", err.Error()
	case strings.Contains(err.Error(), "exceeds"):
		status, code, message = http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", err.Error()
	case strings.Contains(err.Error(), "required"), strings.Contains(err.Error(), "invalid"), strings.Contains(err.Error(), "relationship"), strings.Contains(err.Error(), "tag"):
		status, code, message = http.StatusBadRequest, "VALIDATION_ERROR", err.Error()
	}
	httpserver.WriteError(w, r, status, code, message)
}

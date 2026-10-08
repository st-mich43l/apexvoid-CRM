package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Repository) db(ctx context.Context) querier {
	if tx, ok := database.TransactionFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func contactArgs(c *domain.Contact) ([]byte, error) {
	values := c.CustomValues
	if values == nil {
		values = map[string]any{}
	}
	return json.Marshal(values)
}
func scanContact(row pgx.Row) (*domain.Contact, error) {
	var c domain.Contact
	var values []byte
	err := row.Scan(&c.ID, &c.WorkspaceID, &c.Kind, &c.DisplayName, &c.Email, &c.Phone, &c.Website, &c.Description, &c.Status, &values, &c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.UpdatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(values, &c.CustomValues); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) CreateContact(ctx context.Context, c *domain.Contact) error {
	values, err := contactArgs(c)
	if err != nil {
		return err
	}
	_, err = r.db(ctx).Exec(ctx, `INSERT INTO contacts_contacts(id,workspace_id,kind,display_name,email,phone,website,description,status,custom_values,created_at,updated_at,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, c.ID, c.WorkspaceID, c.Kind, c.DisplayName, c.Email, c.Phone, c.Website, c.Description, c.Status, values, c.CreatedAt, c.UpdatedAt, c.CreatedBy, c.UpdatedBy)
	if isUniqueViolation(err) {
		return domain.ErrDuplicate
	}
	return err
}
func (r *Repository) GetContact(ctx context.Context, workspaceID, id uuid.UUID) (*domain.Contact, error) {
	return scanContact(r.db(ctx).QueryRow(ctx, `SELECT id,workspace_id,kind,display_name,email,phone,website,description,status,custom_values,created_at,updated_at,created_by,updated_by FROM contacts_contacts WHERE workspace_id=$1 AND id=$2`, workspaceID, id))
}
func (r *Repository) ListContacts(ctx context.Context, workspaceID uuid.UUID, f domain.ListFilter) ([]domain.Contact, int, error) {
	where := []string{"c.workspace_id=$1"}
	args := []any{workspaceID}
	next := 2
	if strings.TrimSpace(f.Search) != "" {
		where = append(where, fmt.Sprintf(`(c.display_name ILIKE $%d OR c.email ILIKE $%d OR c.phone ILIKE $%d)`, next, next, next))
		q := "%" + strings.TrimSpace(f.Search) + "%"
		args = append(args, q)
		next++
	}
	if f.Kind != "" {
		where = append(where, fmt.Sprintf("c.kind=$%d", next))
		args = append(args, f.Kind)
		next++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("c.status=$%d", next))
		args = append(args, f.Status)
		next++
	}
	if f.TagID != nil {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM contacts_contact_tags ct WHERE ct.workspace_id=c.workspace_id AND ct.contact_id=c.id AND ct.tag_id=$%d)", next))
		args = append(args, *f.TagID)
		next++
	}
	var total int
	if err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM contacts_contacts c WHERE `+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 25
	}
	if f.Page < 1 {
		f.Page = 1
	}
	sortCol := map[string]string{"name": "c.display_name", "updated": "c.updated_at", "created": "c.created_at"}[f.Sort]
	if sortCol == "" {
		sortCol = "c.display_name"
	}
	direction := "ASC"
	if f.Desc {
		direction = "DESC"
	}
	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	query := fmt.Sprintf(`SELECT c.id,c.workspace_id,c.kind,c.display_name,c.email,c.phone,c.website,c.description,c.status,c.custom_values,c.created_at,c.updated_at,c.created_by,c.updated_by FROM contacts_contacts c WHERE %s ORDER BY %s %s,c.id LIMIT $%d OFFSET $%d`, strings.Join(where, " AND "), sortCol, direction, next, next+1)
	rows, err := r.db(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []domain.Contact{}
	for rows.Next() {
		var c domain.Contact
		var values []byte
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Kind, &c.DisplayName, &c.Email, &c.Phone, &c.Website, &c.Description, &c.Status, &values, &c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.UpdatedBy); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(values, &c.CustomValues); err != nil {
			return nil, 0, err
		}
		result = append(result, c)
	}
	return result, total, rows.Err()
}
func (r *Repository) UpdateContact(ctx context.Context, c *domain.Contact) error {
	values, err := contactArgs(c)
	if err != nil {
		return err
	}
	res, err := r.db(ctx).Exec(ctx, `UPDATE contacts_contacts SET kind=$3,display_name=$4,email=$5,phone=$6,website=$7,description=$8,custom_values=$9,updated_at=$10,updated_by=$11 WHERE workspace_id=$1 AND id=$2`, c.WorkspaceID, c.ID, c.Kind, c.DisplayName, c.Email, c.Phone, c.Website, c.Description, values, c.UpdatedAt, c.UpdatedBy)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *Repository) SetContactStatus(ctx context.Context, workspaceID, id uuid.UUID, status domain.ContactStatus, by uuid.UUID) error {
	res, err := r.db(ctx).Exec(ctx, `UPDATE contacts_contacts SET status=$3,updated_at=$4,updated_by=$5 WHERE workspace_id=$1 AND id=$2`, workspaceID, id, status, time.Now().UTC(), by)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListRelationships(ctx context.Context, w, id uuid.UUID) ([]domain.Relationship, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT rel.id,rel.workspace_id,rel.person_id,rel.company_id,rel.relationship_type,rel.job_title,rel.is_primary,p.display_name,c.display_name FROM contacts_relationships rel JOIN contacts_contacts p ON p.id=rel.person_id AND p.workspace_id=rel.workspace_id JOIN contacts_contacts c ON c.id=rel.company_id AND c.workspace_id=rel.workspace_id WHERE rel.workspace_id=$1 AND (rel.person_id=$2 OR rel.company_id=$2) ORDER BY rel.is_primary DESC,rel.created_at`, w, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Relationship{}
	for rows.Next() {
		var x domain.Relationship
		if err := rows.Scan(&x.ID, &x.WorkspaceID, &x.PersonID, &x.CompanyID, &x.RelationshipType, &x.JobTitle, &x.IsPrimary, &x.PersonName, &x.CompanyName); err != nil {
			return nil, err
		}
		result = append(result, x)
	}
	return result, rows.Err()
}
func (r *Repository) ReplaceRelationships(ctx context.Context, w, id uuid.UUID, items []domain.Relationship) error {
	if _, err := r.db(ctx).Exec(ctx, `DELETE FROM contacts_relationships WHERE workspace_id=$1 AND (person_id=$2 OR company_id=$2)`, w, id); err != nil {
		return err
	}
	for _, x := range items {
		if _, err := r.db(ctx).Exec(ctx, `INSERT INTO contacts_relationships(id,workspace_id,person_id,company_id,relationship_type,job_title,is_primary,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, x.ID, w, x.PersonID, x.CompanyID, x.RelationshipType, x.JobTitle, x.IsPrimary, time.Now().UTC()); isUniqueViolation(err) {
			return domain.ErrDuplicate
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListTags(ctx context.Context, w uuid.UUID, activeOnly bool) ([]domain.Tag, error) {
	q := `SELECT id,workspace_id,name,color,active,created_at,updated_at FROM contacts_tags WHERE workspace_id=$1`
	if activeOnly {
		q += ` AND active=TRUE`
	}
	q += ` ORDER BY name`
	rows, err := r.db(ctx).Query(ctx, q, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Tag{}
	for rows.Next() {
		var x domain.Tag
		if err := rows.Scan(&x.ID, &x.WorkspaceID, &x.Name, &x.Color, &x.Active, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, x)
	}
	return result, rows.Err()
}
func (r *Repository) CreateTag(ctx context.Context, t *domain.Tag) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO contacts_tags(id,workspace_id,name,color,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, t.ID, t.WorkspaceID, t.Name, t.Color, t.Active, t.CreatedAt, t.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicate
	}
	return err
}
func (r *Repository) UpdateTag(ctx context.Context, t *domain.Tag) error {
	res, err := r.db(ctx).Exec(ctx, `UPDATE contacts_tags SET name=$3,color=$4,active=$5,updated_at=$6 WHERE workspace_id=$1 AND id=$2`, t.WorkspaceID, t.ID, t.Name, t.Color, t.Active, t.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicate
	}
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *Repository) ReplaceContactTags(ctx context.Context, w, c uuid.UUID, ids []uuid.UUID) error {
	if _, err := r.db(ctx).Exec(ctx, `DELETE FROM contacts_contact_tags WHERE workspace_id=$1 AND contact_id=$2`, w, c); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := r.db(ctx).Exec(ctx, `INSERT INTO contacts_contact_tags(workspace_id,contact_id,tag_id) VALUES($1,$2,$3)`, w, c, id); err != nil {
			return err
		}
	}
	return nil
}
func (r *Repository) ContactTagIDs(ctx context.Context, w, c uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT tag_id FROM contacts_contact_tags WHERE workspace_id=$1 AND contact_id=$2 ORDER BY tag_id`, w, c)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *Repository) ListNotes(ctx context.Context, w, c uuid.UUID) ([]domain.Note, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT id,workspace_id,contact_id,author_user_id,content,created_at,updated_at FROM contacts_notes WHERE workspace_id=$1 AND contact_id=$2 ORDER BY created_at DESC`, w, c)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Note{}
	for rows.Next() {
		var x domain.Note
		if err := rows.Scan(&x.ID, &x.WorkspaceID, &x.ContactID, &x.AuthorUserID, &x.Content, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) CreateNote(ctx context.Context, n *domain.Note) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO contacts_notes(id,workspace_id,contact_id,author_user_id,content,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, n.ID, n.WorkspaceID, n.ContactID, n.AuthorUserID, n.Content, n.CreatedAt, n.UpdatedAt)
	return err
}
func (r *Repository) UpdateNote(ctx context.Context, n *domain.Note) error {
	res, err := r.db(ctx).Exec(ctx, `UPDATE contacts_notes SET content=$4,updated_at=$5 WHERE workspace_id=$1 AND contact_id=$2 AND id=$3`, n.WorkspaceID, n.ContactID, n.ID, n.Content, n.UpdatedAt)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListActivities(ctx context.Context, w uuid.UUID, c, user *uuid.UUID, status domain.ActivityStatus) ([]domain.Activity, error) {
	where := []string{"workspace_id=$1"}
	args := []any{w}
	if c != nil {
		where = append(where, "related_contact_id=$2")
		args = append(args, *c)
	}
	if user != nil {
		where = append(where, fmt.Sprintf("assigned_user_id=$%d", len(args)+1))
		args = append(args, *user)
	}
	if status != "" {
		where = append(where, fmt.Sprintf("status=$%d", len(args)+1))
		args = append(args, status)
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT id,workspace_id,title,description,activity_type,related_contact_id,assigned_user_id,due_at,status,completed_at,created_at,updated_at FROM contacts_activities WHERE `+strings.Join(where, " AND ")+` ORDER BY due_at NULLS LAST,created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Activity{}
	for rows.Next() {
		var x domain.Activity
		if err := rows.Scan(&x.ID, &x.WorkspaceID, &x.Title, &x.Description, &x.ActivityType, &x.RelatedContactID, &x.AssignedUserID, &x.DueAt, &x.Status, &x.CompletedAt, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) GetActivityByID(ctx context.Context, w, id uuid.UUID) (*domain.Activity, error) {
	var x domain.Activity
	err := r.db(ctx).QueryRow(ctx, `SELECT id,workspace_id,title,description,activity_type,related_contact_id,assigned_user_id,due_at,status,completed_at,created_at,updated_at FROM contacts_activities WHERE workspace_id=$1 AND id=$2`, w, id).Scan(&x.ID, &x.WorkspaceID, &x.Title, &x.Description, &x.ActivityType, &x.RelatedContactID, &x.AssignedUserID, &x.DueAt, &x.Status, &x.CompletedAt, &x.CreatedAt, &x.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}
func (r *Repository) ListActivitiesPage(ctx context.Context, w uuid.UUID, f domain.ActivityListFilter) ([]domain.Activity, int, error) {
	where := []string{"workspace_id=$1"}
	args := []any{w}
	next := 2
	if f.ContactID != nil {
		where = append(where, fmt.Sprintf("related_contact_id=$%d", next))
		args = append(args, *f.ContactID)
		next++
	}
	if f.AssignedUserID != nil {
		where = append(where, fmt.Sprintf("assigned_user_id=$%d", next))
		args = append(args, *f.AssignedUserID)
		next++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status=$%d", next))
		args = append(args, f.Status)
		next++
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM contacts_activities WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 25
	}
	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	query := fmt.Sprintf(`SELECT id,workspace_id,title,description,activity_type,related_contact_id,assigned_user_id,due_at,status,completed_at,created_at,updated_at FROM contacts_activities WHERE %s ORDER BY due_at NULLS LAST,created_at DESC,id LIMIT $%d OFFSET $%d`, whereSQL, next, next+1)
	rows, err := r.db(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Activity{}
	for rows.Next() {
		var x domain.Activity
		if err := rows.Scan(&x.ID, &x.WorkspaceID, &x.Title, &x.Description, &x.ActivityType, &x.RelatedContactID, &x.AssignedUserID, &x.DueAt, &x.Status, &x.CompletedAt, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, x)
	}
	return out, total, rows.Err()
}
func (r *Repository) CreateActivity(ctx context.Context, a *domain.Activity) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO contacts_activities(id,workspace_id,title,description,activity_type,related_contact_id,assigned_user_id,due_at,status,completed_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, a.ID, a.WorkspaceID, a.Title, a.Description, a.ActivityType, a.RelatedContactID, a.AssignedUserID, a.DueAt, a.Status, a.CompletedAt, a.CreatedAt, a.UpdatedAt)
	return err
}
func (r *Repository) UpdateActivity(ctx context.Context, a *domain.Activity) error {
	res, err := r.db(ctx).Exec(ctx, `UPDATE contacts_activities SET title=$3,description=$4,activity_type=$5,related_contact_id=$6,assigned_user_id=$7,due_at=$8,status=$9,completed_at=$10,updated_at=$11 WHERE workspace_id=$1 AND id=$2`, a.WorkspaceID, a.ID, a.Title, a.Description, a.ActivityType, a.RelatedContactID, a.AssignedUserID, a.DueAt, a.Status, a.CompletedAt, a.UpdatedAt)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListAttachments(ctx context.Context, w, c uuid.UUID) ([]domain.Attachment, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT id,workspace_id,contact_id,file_name,size,content_type,uploaded_by,created_at FROM contacts_attachments WHERE workspace_id=$1 AND contact_id=$2 ORDER BY created_at DESC`, w, c)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Attachment{}
	for rows.Next() {
		var x domain.Attachment
		if err := rows.Scan(&x.ID, &x.WorkspaceID, &x.ContactID, &x.FileName, &x.Size, &x.ContentType, &x.UploadedBy, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) CreateAttachment(ctx context.Context, a *domain.Attachment, key string) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO contacts_attachments(id,workspace_id,contact_id,file_name,storage_key,size,content_type,uploaded_by,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, a.ID, a.WorkspaceID, a.ContactID, a.FileName, key, a.Size, a.ContentType, a.UploadedBy, a.CreatedAt)
	return err
}
func (r *Repository) GetAttachment(ctx context.Context, w, c, id uuid.UUID) (domain.Attachment, string, error) {
	var a domain.Attachment
	var key string
	err := r.db(ctx).QueryRow(ctx, `SELECT id,workspace_id,contact_id,file_name,storage_key,size,content_type,uploaded_by,created_at FROM contacts_attachments WHERE workspace_id=$1 AND contact_id=$2 AND id=$3`, w, c, id).Scan(&a.ID, &a.WorkspaceID, &a.ContactID, &a.FileName, &key, &a.Size, &a.ContentType, &a.UploadedBy, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, "", domain.ErrNotFound
	}
	return a, key, err
}
func (r *Repository) DeleteAttachment(ctx context.Context, w, c, id uuid.UUID) error {
	res, err := r.db(ctx).Exec(ctx, `DELETE FROM contacts_attachments WHERE workspace_id=$1 AND contact_id=$2 AND id=$3`, w, c, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListCustomFields(ctx context.Context, w uuid.UUID, activeOnly bool) ([]domain.CustomFieldDefinition, error) {
	q := `SELECT id,workspace_id,entity,field_key,label,field_type,description,required,options,display_order,active,created_at,updated_at FROM contacts_custom_field_definitions WHERE workspace_id=$1`
	if activeOnly {
		q += ` AND active=TRUE`
	}
	q += ` ORDER BY display_order,label`
	rows, err := r.db(ctx).Query(ctx, q, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CustomFieldDefinition{}
	for rows.Next() {
		var x domain.CustomFieldDefinition
		var options []byte
		if err := rows.Scan(&x.ID, &x.WorkspaceID, &x.Entity, &x.Key, &x.Label, &x.Type, &x.Description, &x.Required, &options, &x.DisplayOrder, &x.Active, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(options, &x.Options); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) CreateCustomField(ctx context.Context, f *domain.CustomFieldDefinition) error {
	options, _ := json.Marshal(f.Options)
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO contacts_custom_field_definitions(id,workspace_id,entity,field_key,label,field_type,description,required,options,display_order,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, f.ID, f.WorkspaceID, f.Entity, f.Key, f.Label, f.Type, f.Description, f.Required, options, f.DisplayOrder, f.Active, f.CreatedAt, f.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicate
	}
	return err
}
func (r *Repository) UpdateCustomField(ctx context.Context, f *domain.CustomFieldDefinition) error {
	options, _ := json.Marshal(f.Options)
	res, err := r.db(ctx).Exec(ctx, `UPDATE contacts_custom_field_definitions SET label=$3,field_type=$4,description=$5,required=$6,options=$7,display_order=$8,active=$9,updated_at=$10 WHERE workspace_id=$1 AND id=$2`, f.WorkspaceID, f.ID, f.Label, f.Type, f.Description, f.Required, options, f.DisplayOrder, f.Active, f.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicate
	}
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *Repository) CountContactsWithCustomFieldValues(ctx context.Context, w uuid.UUID, key string) (int, error) {
	var count int
	err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM contacts_contacts WHERE workspace_id=$1 AND custom_values ? $2 AND custom_values->>$2 IS NOT NULL`, w, key).Scan(&count)
	return count, err
}
func (r *Repository) CountContactsUsingCustomFieldOptions(ctx context.Context, w uuid.UUID, key string, options []string) (int, error) {
	var count int
	err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM contacts_contacts WHERE workspace_id=$1 AND custom_values->>$2 IS NOT NULL AND NOT (custom_values->>$2 = ANY($3::text[]))`, w, key, options).Scan(&count)
	return count, err
}
func (r *Repository) CountContactsMissingCustomField(ctx context.Context, w uuid.UUID, key string) (int, error) {
	var count int
	err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM contacts_contacts WHERE workspace_id=$1 AND (NOT (custom_values ? $2) OR custom_values->>$2 IS NULL OR custom_values->>$2='')`, w, key).Scan(&count)
	return count, err
}
func (r *Repository) IsActiveWorkspaceMember(ctx context.Context, w, user uuid.UUID) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_memberships m JOIN workspace_workspaces ws ON ws.id=m.workspace_id JOIN organization_organizations o ON o.id=ws.organization_id JOIN users_users u ON u.id=m.user_id WHERE m.workspace_id=$1 AND m.user_id=$2 AND m.status='active' AND ws.status='active' AND o.status='active' AND u.status='active')`, w, user).Scan(&ok)
	return ok, err
}
func isUniqueViolation(err error) bool {
	var e *pgconn.PgError
	return errors.As(err, &e) && e.Code == "23505"
}

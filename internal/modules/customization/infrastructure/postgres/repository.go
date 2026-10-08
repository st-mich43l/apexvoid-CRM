package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/domain"
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

func (r *Repository) GetField(ctx context.Context, workspaceID, id uuid.UUID) (domain.RuntimeField, error) {
	return scanField(r.db(ctx).QueryRow(ctx, "SELECT id,workspace_id,entity_name,field_key,label,field_type,description,required,default_value,options,visible,display_order,section_id,active,created_at,updated_at FROM customization_field_definitions WHERE workspace_id=$1 AND id=$2 FOR UPDATE", workspaceID, id))
}

func (r *Repository) ListFields(ctx context.Context, workspaceID uuid.UUID, entity string, activeOnly bool) ([]domain.RuntimeField, error) {
	query := `SELECT id,workspace_id,entity_name,field_key,label,field_type,description,required,default_value,options,visible,display_order,section_id,active,created_at,updated_at FROM customization_field_definitions WHERE workspace_id=$1 AND entity_name=$2`
	if activeOnly {
		query += ` AND active=TRUE`
	}
	query += ` ORDER BY display_order,field_key`
	rows, err := r.db(ctx).Query(ctx, query, workspaceID, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RuntimeField{}
	for rows.Next() {
		item, scanErr := scanField(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateField(ctx context.Context, item *domain.RuntimeField) error {
	defaultValue, options, err := fieldJSON(item)
	if err != nil {
		return err
	}
	_, err = r.db(ctx).Exec(ctx, `INSERT INTO customization_field_definitions(id,workspace_id,entity_name,field_key,label,field_type,description,required,default_value,options,visible,display_order,section_id,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, item.ID, item.WorkspaceID, item.Entity, item.Key, item.Label, item.Type, item.Description, item.Required, defaultValue, options, item.Visible, item.DisplayOrder, item.SectionID, item.Active, item.CreatedAt, item.UpdatedAt)
	return uniqueError(err)
}

func (r *Repository) UpdateField(ctx context.Context, item *domain.RuntimeField) error {
	defaultValue, options, err := fieldJSON(item)
	if err != nil {
		return err
	}
	result, err := r.db(ctx).Exec(ctx, `UPDATE customization_field_definitions SET label=$4,field_type=$5,description=$6,required=$7,default_value=$8,options=$9,visible=$10,display_order=$11,section_id=$12,active=$13,updated_at=$14 WHERE workspace_id=$1 AND entity_name=$2 AND id=$3 AND field_key=$15`, item.WorkspaceID, item.Entity, item.ID, item.Label, item.Type, item.Description, item.Required, defaultValue, options, item.Visible, item.DisplayOrder, item.SectionID, item.Active, item.UpdatedAt, item.Key)
	if err != nil {
		return uniqueError(err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListSections(ctx context.Context, workspaceID uuid.UUID, entity string) ([]domain.FormSection, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT id,workspace_id,entity_name,name,description,display_order,created_at,updated_at FROM customization_form_sections WHERE workspace_id=$1 AND entity_name=$2 ORDER BY display_order,name`, workspaceID, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.FormSection{}
	for rows.Next() {
		var item domain.FormSection
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Entity, &item.Name, &item.Description, &item.DisplayOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateSection(ctx context.Context, item *domain.FormSection) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO customization_form_sections(id,workspace_id,entity_name,name,description,display_order,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, item.ID, item.WorkspaceID, item.Entity, item.Name, item.Description, item.DisplayOrder, item.CreatedAt, item.UpdatedAt)
	return uniqueError(err)
}

func (r *Repository) UpdateSection(ctx context.Context, item *domain.FormSection) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE customization_form_sections SET name=$4,description=$5,display_order=$6,updated_at=$7 WHERE workspace_id=$1 AND entity_name=$2 AND id=$3`, item.WorkspaceID, item.Entity, item.ID, item.Name, item.Description, item.DisplayOrder, item.UpdatedAt)
	if err != nil {
		return uniqueError(err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListViews(ctx context.Context, workspaceID, userID uuid.UUID, entity string) ([]domain.SavedView, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT id,workspace_id,entity_name,owner_user_id,name,shared,filters,columns,sort_field,sort_direction,created_at,updated_at FROM customization_saved_views WHERE workspace_id=$1 AND entity_name=$2 AND (shared=TRUE OR owner_user_id=$3) ORDER BY shared DESC,name`, workspaceID, entity, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SavedView{}
	for rows.Next() {
		var item domain.SavedView
		var filters, columns []byte
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Entity, &item.OwnerUserID, &item.Name, &item.Shared, &filters, &columns, &item.SortField, &item.SortDirection, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(filters, &item.Filters); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(columns, &item.Columns); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateView(ctx context.Context, item *domain.SavedView) error {
	filters, columns, err := viewJSON(item)
	if err != nil {
		return err
	}
	_, err = r.db(ctx).Exec(ctx, `INSERT INTO customization_saved_views(id,workspace_id,entity_name,owner_user_id,name,shared,filters,columns,sort_field,sort_direction,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, item.ID, item.WorkspaceID, item.Entity, item.OwnerUserID, item.Name, item.Shared, filters, columns, item.SortField, item.SortDirection, item.CreatedAt, item.UpdatedAt)
	return uniqueError(err)
}

func (r *Repository) UpdateView(ctx context.Context, item *domain.SavedView) error {
	filters, columns, err := viewJSON(item)
	if err != nil {
		return err
	}
	result, err := r.db(ctx).Exec(ctx, `UPDATE customization_saved_views SET name=$5,shared=$6,filters=$7,columns=$8,sort_field=$9,sort_direction=$10,updated_at=$11 WHERE workspace_id=$1 AND entity_name=$2 AND id=$3 AND owner_user_id=$4`, item.WorkspaceID, item.Entity, item.ID, item.OwnerUserID, item.Name, item.Shared, filters, columns, item.SortField, item.SortDirection, item.UpdatedAt)
	if err != nil {
		return uniqueError(err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteView(ctx context.Context, workspaceID uuid.UUID, entity string, id, ownerID uuid.UUID) error {
	result, err := r.db(ctx).Exec(ctx, `DELETE FROM customization_saved_views WHERE workspace_id=$1 AND entity_name=$2 AND id=$3 AND owner_user_id=$4`, workspaceID, entity, id, ownerID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanField(row pgx.Row) (domain.RuntimeField, error) {
	var item domain.RuntimeField
	var defaultValue, options []byte
	err := row.Scan(&item.ID, &item.WorkspaceID, &item.Entity, &item.Key, &item.Label, &item.Type, &item.Description, &item.Required, &defaultValue, &options, &item.Visible, &item.DisplayOrder, &item.SectionID, &item.Active, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RuntimeField{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RuntimeField{}, err
	}
	if len(defaultValue) > 0 && string(defaultValue) != "null" {
		if err := json.Unmarshal(defaultValue, &item.DefaultValue); err != nil {
			return domain.RuntimeField{}, err
		}
	}
	if err := json.Unmarshal(options, &item.Options); err != nil {
		return domain.RuntimeField{}, err
	}
	return item, nil
}

func fieldJSON(item *domain.RuntimeField) ([]byte, []byte, error) {
	defaultValue, err := json.Marshal(item.DefaultValue)
	if err != nil {
		return nil, nil, err
	}
	options, err := json.Marshal(item.Options)
	if err != nil {
		return nil, nil, err
	}
	return defaultValue, options, nil
}

func viewJSON(item *domain.SavedView) ([]byte, []byte, error) {
	filters, err := json.Marshal(item.Filters)
	if err != nil {
		return nil, nil, err
	}
	columns, err := json.Marshal(item.Columns)
	if err != nil {
		return nil, nil, err
	}
	return filters, columns, nil
}

func uniqueError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}

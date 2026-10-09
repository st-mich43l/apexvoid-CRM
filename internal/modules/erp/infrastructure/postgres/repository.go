package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const productColumns = "id,workspace_id,sku,name,description,kind,unit,unit_price::text,currency,status,version,created_at,updated_at"

func scan(row pgx.Row) (domain.Product, error) {
	var p domain.Product
	err := row.Scan(&p.ID, &p.WorkspaceID, &p.SKU, &p.Name, &p.Description, &p.Kind, &p.Unit, &p.UnitPrice, &p.Currency, &p.Status, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	return p, mapError(err)
}
func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "23514" || pgErr.Code == "23503") {
		return domain.ErrConflict
	}
	return err
}
func (r *Repository) Create(ctx context.Context, workspaceID, actorID uuid.UUID, p domain.Input) (domain.Product, error) {
	id := uuid.New()
	return scan(r.pool.QueryRow(ctx, `INSERT INTO erp_products(id,workspace_id,sku,name,description,kind,unit,unit_price,currency,created_by,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8::numeric,$9,$10,$10) RETURNING `+productColumns,
		id, workspaceID, p.SKU, p.Name, p.Description, p.Kind, p.Unit, p.UnitPrice, p.Currency, actorID))
}
func (r *Repository) List(ctx context.Context, workspaceID uuid.UUID, search string, includeArchived bool, page, limit int) ([]domain.Product, int, error) {
	filter := strings.TrimSpace(search)
	if len(filter) > 200 {
		filter = filter[:200]
	}
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM erp_products WHERE workspace_id=$1 AND ($2 OR status='active') AND ($3='' OR name ILIKE '%'||$3||'%' OR sku ILIKE '%'||$3||'%')`, workspaceID, includeArchived, filter).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+productColumns+` FROM erp_products WHERE workspace_id=$1 AND ($2 OR status='active') AND ($3='' OR name ILIKE '%'||$3||'%' OR sku ILIKE '%'||$3||'%') ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`, workspaceID, includeArchived, filter, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]domain.Product, 0)
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
func (r *Repository) Get(ctx context.Context, workspaceID, id uuid.UUID) (domain.Product, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+productColumns+` FROM erp_products WHERE workspace_id=$1 AND id=$2`, workspaceID, id))
}
func (r *Repository) Update(ctx context.Context, workspaceID, actorID, id uuid.UUID, version int, p domain.Input) (domain.Product, error) {
	item, err := scan(r.pool.QueryRow(ctx, `UPDATE erp_products SET sku=$4,name=$5,description=$6,kind=$7,unit=$8,unit_price=$9::numeric,currency=$10,version=version+1,updated_by=$11,updated_at=NOW() WHERE workspace_id=$1 AND id=$2 AND version=$3 RETURNING `+productColumns,
		workspaceID, id, version, p.SKU, p.Name, p.Description, p.Kind, p.Unit, p.UnitPrice, p.Currency, actorID))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Product{}, domain.ErrConflict
	}
	return item, err
}
func (r *Repository) SetArchived(ctx context.Context, workspaceID, actorID, id uuid.UUID, version int, archived bool) (domain.Product, error) {
	status := "active"
	if archived {
		status = "archived"
	}
	item, err := scan(r.pool.QueryRow(ctx, `UPDATE erp_products SET status=$4,version=version+1,updated_by=$5,updated_at=NOW() WHERE workspace_id=$1 AND id=$2 AND version=$3 RETURNING `+productColumns, workspaceID, id, version, status, actorID))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Product{}, domain.ErrConflict
	}
	return item, err
}

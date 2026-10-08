package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool} }

type db interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Repository) q(ctx context.Context) db {
	if tx, ok := database.TransactionFromContext(ctx); ok {
		return tx
	}
	return r.pool
}
func (r *Repository) LockWorkspace(ctx context.Context, w uuid.UUID) error {
	_, e := r.q(ctx).Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", w.String())
	return e
}
func scanP(row pgx.Row) (*domain.Pipeline, error) {
	var p domain.Pipeline
	e := row.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Slug, &p.Description, &p.Color, &p.Status, &p.Default, &p.DisplayOrder, &p.CreatedBy, &p.UpdatedBy, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &p, e
}
func (r *Repository) ListPipelines(ctx context.Context, w uuid.UUID, archived bool) ([]domain.Pipeline, error) {
	q := "SELECT id,workspace_id,name,slug,description,color,status,is_default,display_order,created_by,updated_by,version,created_at,updated_at FROM crm_pipelines WHERE workspace_id=$1"
	if !archived {
		q += " AND status='active'"
	}
	q += " ORDER BY display_order,name"
	rows, e := r.q(ctx).Query(ctx, q, w)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Pipeline{}
	for rows.Next() {
		p, e := scanP(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}
func (r *Repository) GetPipeline(ctx context.Context, w, id uuid.UUID) (*domain.Pipeline, error) {
	return scanP(r.q(ctx).QueryRow(ctx, "SELECT id,workspace_id,name,slug,description,color,status,is_default,display_order,created_by,updated_by,version,created_at,updated_at FROM crm_pipelines WHERE workspace_id=$1 AND id=$2", w, id))
}
func (r *Repository) CreatePipeline(ctx context.Context, p *domain.Pipeline) error {
	_, e := r.q(ctx).Exec(ctx, "INSERT INTO crm_pipelines(id,workspace_id,name,slug,description,color,status,is_default,display_order,created_by,updated_by,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)", p.ID, p.WorkspaceID, p.Name, p.Slug, p.Description, p.Color, p.Status, p.Default, p.DisplayOrder, p.CreatedBy, p.UpdatedBy, p.Version, p.CreatedAt, p.UpdatedAt)
	return mapErr(e)
}
func (r *Repository) UpdatePipeline(ctx context.Context, p *domain.Pipeline, v int) error {
	x, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipelines SET name=$3,description=$4,color=$5,display_order=$6,updated_by=$7,version=version+1,updated_at=$8 WHERE workspace_id=$1 AND id=$2 AND version=$9", p.WorkspaceID, p.ID, p.Name, p.Description, p.Color, p.DisplayOrder, p.UpdatedBy, p.UpdatedAt, v)
	if e != nil {
		return mapErr(e)
	}
	if x.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	p.Version = v + 1
	return nil
}
func (r *Repository) SetDefault(ctx context.Context, w, id uuid.UUID) error {
	if _, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipelines SET is_default=FALSE WHERE workspace_id=$1 AND is_default", w); e != nil {
		return e
	}
	x, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipelines SET is_default=TRUE,version=version+1,updated_at=NOW() WHERE workspace_id=$1 AND id=$2 AND status='active'", w, id)
	if e != nil {
		return e
	}
	if x.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func scanS(row pgx.Row) (*domain.Stage, error) {
	var s domain.Stage
	e := row.Scan(&s.ID, &s.WorkspaceID, &s.PipelineID, &s.Key, &s.Name, &s.Description, &s.Color, &s.Position, &s.Category, &s.Probability, &s.Active, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &s, e
}
func (r *Repository) ListStages(ctx context.Context, w, p uuid.UUID, all bool) ([]domain.Stage, error) {
	q := "SELECT id,workspace_id,pipeline_id,stage_key,name,description,color,position,category,probability,active,version,created_at,updated_at FROM crm_pipeline_stages WHERE workspace_id=$1 AND pipeline_id=$2"
	if !all {
		q += " AND active"
	}
	q += " ORDER BY position,id"
	rows, e := r.q(ctx).Query(ctx, q, w, p)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Stage{}
	for rows.Next() {
		s, e := scanS(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}
func (r *Repository) CreateStages(ctx context.Context, items []domain.Stage) error {
	for _, s := range items {
		_, e := r.q(ctx).Exec(ctx, "INSERT INTO crm_pipeline_stages(id,workspace_id,pipeline_id,stage_key,name,description,color,position,category,probability,active,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)", s.ID, s.WorkspaceID, s.PipelineID, s.Key, s.Name, s.Description, s.Color, s.Position, s.Category, s.Probability, s.Active, s.Version, s.CreatedAt, s.UpdatedAt)
		if e != nil {
			return mapErr(e)
		}
	}
	return nil
}
func (r *Repository) UpdateStage(ctx context.Context, s *domain.Stage, v int) error {
	x, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipeline_stages SET name=$4,description=$5,color=$6,probability=$7,version=version+1,updated_at=$8 WHERE workspace_id=$1 AND pipeline_id=$2 AND id=$3 AND version=$9", s.WorkspaceID, s.PipelineID, s.ID, s.Name, s.Description, s.Color, s.Probability, s.UpdatedAt, v)
	if e != nil {
		return mapErr(e)
	}
	if x.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	s.Version = v + 1
	return nil
}
func (r *Repository) SetStagesActive(ctx context.Context, w, p uuid.UUID, ids []uuid.UUID, a bool) error {
	for _, id := range ids {
		x, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipeline_stages SET active=$4,version=version+1,updated_at=NOW() WHERE workspace_id=$1 AND pipeline_id=$2 AND id=$3", w, p, id, a)
		if e != nil {
			return e
		}
		if x.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
	}
	return nil
}
func (r *Repository) SetTemporaryStagePositions(ctx context.Context, w, p uuid.UUID) error {
	_, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipeline_stages SET position=position+1000000 WHERE workspace_id=$1 AND pipeline_id=$2 AND active", w, p)
	return e
}
func (r *Repository) SetStagePositions(ctx context.Context, w, p uuid.UUID, ids []uuid.UUID) error {
	for i, id := range ids {
		_, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipeline_stages SET position=$4,version=version+1,updated_at=NOW() WHERE workspace_id=$1 AND pipeline_id=$2 AND id=$3", w, p, id, i)
		if e != nil {
			return e
		}
	}
	return nil
}
func mapErr(e error) error {
	var p *pgconn.PgError
	if errors.As(e, &p) && p.Code == "23505" {
		return domain.ErrConflict
	}
	return e
}

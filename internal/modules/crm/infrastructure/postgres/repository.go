package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
	"strings"
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
func (r *Repository) SetPipelineStatus(ctx context.Context, w, id uuid.UUID, status domain.PipelineStatus, version int) error {
	x, e := r.q(ctx).Exec(ctx, "UPDATE crm_pipelines SET status=$3,is_default=CASE WHEN $3='archived' THEN FALSE ELSE is_default END,version=version+1,updated_at=NOW() WHERE workspace_id=$1 AND id=$2 AND version=$4", w, id, status, version)
	if e != nil {
		return mapErr(e)
	}
	if x.RowsAffected() == 0 {
		return domain.ErrConflict
	}
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
func (r *Repository) CountOpenOpportunities(ctx context.Context, w, pipelineID uuid.UUID, stageID *uuid.UUID) (int, error) {
	var count int
	query := "SELECT count(*) FROM crm_opportunities WHERE workspace_id=$1 AND pipeline_id=$2 AND outcome='open'"
	args := []any{w, pipelineID}
	if stageID != nil {
		query += " AND stage_id=$3"
		args = append(args, *stageID)
	}
	return count, r.q(ctx).QueryRow(ctx, query, args...).Scan(&count)
}
func (r *Repository) SetTemporaryStagePositions(ctx context.Context, w, p uuid.UUID) error {
	var maxPosition, count int
	if e := r.q(ctx).QueryRow(ctx, "SELECT COALESCE(MAX(position),-1),COUNT(*) FROM crm_pipeline_stages WHERE workspace_id=$1 AND pipeline_id=$2 AND active", w, p).Scan(&maxPosition, &count); e != nil {
		return e
	}
	rows, e := r.q(ctx).Query(ctx, "SELECT id FROM crm_pipeline_stages WHERE workspace_id=$1 AND pipeline_id=$2 AND active ORDER BY position,id", w, p)
	if e != nil {
		return e
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if e = rows.Scan(&id); e != nil {
			return e
		}
		ids = append(ids, id)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	base := maxPosition + count + 1
	for index, id := range ids {
		if _, e = r.q(ctx).Exec(ctx, "UPDATE crm_pipeline_stages SET position=$3 WHERE workspace_id=$1 AND pipeline_id=$2 AND id=$4", w, p, base+index, id); e != nil {
			return e
		}
	}
	return nil
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

const leadColumns = "id,workspace_id,title,description,contact_name,company_name,email,phone,source,assigned_user_id,contact_id,status,disqualification_reason,custom_values,converted_opportunity_id,converted_at,created_by,updated_by,created_at,updated_at,version"
const opportunityColumns = "id,workspace_id,title,description,pipeline_id,stage_id,contact_id,company_id,assigned_user_id,expected_revenue::text,currency,expected_close_date,outcome,loss_reason,custom_values,original_lead_id,created_by,updated_by,created_at,updated_at,closed_at,version"
const opportunityInsertColumns = "id,workspace_id,title,description,pipeline_id,stage_id,contact_id,company_id,assigned_user_id,expected_revenue,currency,expected_close_date,outcome,loss_reason,custom_values,original_lead_id,created_by,updated_by,created_at,updated_at,closed_at,version"

func scanLead(row pgx.Row) (*domain.Lead, error) {
	var item domain.Lead
	var values []byte
	err := row.Scan(&item.ID, &item.WorkspaceID, &item.Title, &item.Description, &item.ContactName, &item.CompanyName, &item.Email, &item.Phone, &item.Source, &item.AssignedUserID, &item.ContactID, &item.Status, &item.DisqualificationReason, &values, &item.ConvertedOpportunityID, &item.ConvertedAt, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt, &item.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(values, &item.CustomValues); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanOpportunity(row pgx.Row) (*domain.Opportunity, error) {
	var item domain.Opportunity
	var values []byte
	err := row.Scan(&item.ID, &item.WorkspaceID, &item.Title, &item.Description, &item.PipelineID, &item.StageID, &item.ContactID, &item.CompanyID, &item.AssignedUserID, &item.ExpectedRevenue, &item.Currency, &item.ExpectedCloseDate, &item.Outcome, &item.LossReason, &values, &item.OriginalLeadID, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt, &item.ClosedAt, &item.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(values, &item.CustomValues); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetLead(ctx context.Context, w, id uuid.UUID) (*domain.Lead, error) {
	return scanLead(r.q(ctx).QueryRow(ctx, "SELECT "+leadColumns+" FROM crm_leads WHERE workspace_id=$1 AND id=$2", w, id))
}
func (r *Repository) GetLeadForUpdate(ctx context.Context, w, id uuid.UUID) (*domain.Lead, error) {
	return scanLead(r.q(ctx).QueryRow(ctx, "SELECT "+leadColumns+" FROM crm_leads WHERE workspace_id=$1 AND id=$2 FOR UPDATE", w, id))
}
func (r *Repository) ListLeads(ctx context.Context, w uuid.UUID, f domain.LeadFilter) ([]domain.Lead, int, error) {
	where, args := []string{"workspace_id=$1"}, []any{w}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, clause+"=$"+fmt.Sprint(len(args)))
	}
	if f.Status != "" {
		add("status", f.Status)
	}
	if f.OwnerID != nil {
		add("assigned_user_id", *f.OwnerID)
	}
	if strings.TrimSpace(f.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(f.Search)+"%")
		where = append(where, "(title ILIKE $"+fmt.Sprint(len(args))+" OR contact_name ILIKE $"+fmt.Sprint(len(args))+" OR company_name ILIKE $"+fmt.Sprint(len(args))+" OR email ILIKE $"+fmt.Sprint(len(args))+")")
	}
	if err := applyViewConditions(&where, &args, f.Conditions, "crm.lead"); err != nil { return nil, 0, err }
    filter := strings.Join(where, " AND ")
	var total int
	if err := r.q(ctx).QueryRow(ctx, "SELECT count(*) FROM crm_leads WHERE "+filter, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sort := map[string]string{"title": "title", "created_at": "created_at", "updated_at": "updated_at", "status": "status"}[f.Sort]
    if f.SortCustom {
        args = append(args, f.Sort)
        sort = viewExpression("custom_values->>$"+fmt.Sprint(len(args)), f.SortType)
    } else if f.Sort != "" && sort == "" {
        return nil, 0, fmt.Errorf("unsupported lead sort field %q", f.Sort)
    }
    if sort == "" { sort = "created_at" }
	direction := "ASC"
	if f.Desc {
		direction = "DESC"
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 25
	}
	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	rows, err := r.q(ctx).Query(ctx, "SELECT "+leadColumns+" FROM crm_leads WHERE "+filter+" ORDER BY "+sort+" "+direction+", id ASC LIMIT $"+fmt.Sprint(len(args)-1)+" OFFSET $"+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []domain.Lead{}
	for rows.Next() {
		item, e := scanLead(rows)
		if e != nil {
			return nil, 0, e
		}
		result = append(result, *item)
	}
	return result, total, rows.Err()
}
func (r *Repository) CreateLead(ctx context.Context, item *domain.Lead) error {
	values, err := json.Marshal(item.CustomValues)
	if err != nil {
		return err
	}
	_, err = r.q(ctx).Exec(ctx, "INSERT INTO crm_leads("+leadColumns+") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)", item.ID, item.WorkspaceID, item.Title, item.Description, item.ContactName, item.CompanyName, item.Email, item.Phone, item.Source, item.AssignedUserID, item.ContactID, item.Status, item.DisqualificationReason, values, item.ConvertedOpportunityID, item.ConvertedAt, item.CreatedBy, item.UpdatedBy, item.CreatedAt, item.UpdatedAt, item.Version)
	return mapErr(err)
}
func (r *Repository) UpdateLead(ctx context.Context, item *domain.Lead, version int) error {
	values, err := json.Marshal(item.CustomValues)
	if err != nil {
		return err
	}
	x, err := r.q(ctx).Exec(ctx, "UPDATE crm_leads SET title=$3,description=$4,contact_name=$5,company_name=$6,email=$7,phone=$8,source=$9,assigned_user_id=$10,contact_id=$11,status=$12,disqualification_reason=$13,custom_values=$14,converted_opportunity_id=$15,converted_at=$16,updated_by=$17,updated_at=$18,version=version+1 WHERE workspace_id=$1 AND id=$2 AND version=$19", item.WorkspaceID, item.ID, item.Title, item.Description, item.ContactName, item.CompanyName, item.Email, item.Phone, item.Source, item.AssignedUserID, item.ContactID, item.Status, item.DisqualificationReason, values, item.ConvertedOpportunityID, item.ConvertedAt, item.UpdatedBy, item.UpdatedAt, version)
	if err != nil {
		return mapErr(err)
	}
	if x.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	item.Version = version + 1
	return nil
}
func (r *Repository) GetOpportunity(ctx context.Context, w, id uuid.UUID) (*domain.Opportunity, error) {
	return scanOpportunity(r.q(ctx).QueryRow(ctx, "SELECT "+opportunityColumns+" FROM crm_opportunities WHERE workspace_id=$1 AND id=$2", w, id))
}
func (r *Repository) GetOpportunityForUpdate(ctx context.Context, w, id uuid.UUID) (*domain.Opportunity, error) {
	return scanOpportunity(r.q(ctx).QueryRow(ctx, "SELECT "+opportunityColumns+" FROM crm_opportunities WHERE workspace_id=$1 AND id=$2 FOR UPDATE", w, id))
}
func (r *Repository) ListOpportunities(ctx context.Context, w uuid.UUID, f domain.OpportunityFilter) ([]domain.Opportunity, int, error) {
	where, args := []string{"workspace_id=$1"}, []any{w}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, clause+"=$"+fmt.Sprint(len(args)))
	}
	if f.PipelineID != nil {
		add("pipeline_id", *f.PipelineID)
	}
	if f.StageID != nil {
		add("stage_id", *f.StageID)
	}
	if f.OwnerID != nil {
		add("assigned_user_id", *f.OwnerID)
	}
	if f.Outcome != "" {
		add("outcome", f.Outcome)
	}
	if strings.TrimSpace(f.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(f.Search)+"%")
		where = append(where, "title ILIKE $"+fmt.Sprint(len(args)))
	}
	if err := applyViewConditions(&where, &args, f.Conditions, "crm.opportunity"); err != nil { return nil, 0, err }
    filter := strings.Join(where, " AND ")
	var total int
	if err := r.q(ctx).QueryRow(ctx, "SELECT count(*) FROM crm_opportunities WHERE "+filter, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sort := map[string]string{"title": "title", "created_at": "created_at", "updated_at": "updated_at", "expected_revenue": "expected_revenue", "expected_close_date": "expected_close_date", "outcome": "outcome", "currency": "currency"}[f.Sort]
    if f.SortCustom {
        args = append(args, f.Sort)
        sort = viewExpression("custom_values->>$"+fmt.Sprint(len(args)), f.SortType)
    } else if f.Sort != "" && sort == "" {
        return nil, 0, fmt.Errorf("unsupported opportunity sort field %q", f.Sort)
    }
    if sort == "" { sort = "created_at" }
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 25
	}
	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	rows, err := r.q(ctx).Query(ctx, "SELECT "+opportunityColumns+" FROM crm_opportunities WHERE "+filter+" ORDER BY "+sort+" "+dir+", id ASC LIMIT $"+fmt.Sprint(len(args)-1)+" OFFSET $"+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []domain.Opportunity{}
	for rows.Next() {
		item, e := scanOpportunity(rows)
		if e != nil {
			return nil, 0, e
		}
		result = append(result, *item)
	}
	return result, total, rows.Err()
}
func (r *Repository) CreateOpportunity(ctx context.Context, item *domain.Opportunity) error {
	values, err := json.Marshal(item.CustomValues)
	if err != nil {
		return err
	}
	_, err = r.q(ctx).Exec(ctx, "INSERT INTO crm_opportunities("+opportunityInsertColumns+") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::numeric,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)", item.ID, item.WorkspaceID, item.Title, item.Description, item.PipelineID, item.StageID, item.ContactID, item.CompanyID, item.AssignedUserID, item.ExpectedRevenue, item.Currency, item.ExpectedCloseDate, item.Outcome, item.LossReason, values, item.OriginalLeadID, item.CreatedBy, item.UpdatedBy, item.CreatedAt, item.UpdatedAt, item.ClosedAt, item.Version)
	return mapErr(err)
}
func (r *Repository) UpdateOpportunity(ctx context.Context, item *domain.Opportunity, version int) error {
	values, err := json.Marshal(item.CustomValues)
	if err != nil {
		return err
	}
	x, err := r.q(ctx).Exec(ctx, "UPDATE crm_opportunities SET title=$3,description=$4,pipeline_id=$5,stage_id=$6,contact_id=$7,company_id=$8,assigned_user_id=$9,expected_revenue=$10::numeric,currency=$11,expected_close_date=$12,outcome=$13,loss_reason=$14,custom_values=$15,closed_at=$16,updated_by=$17,updated_at=$18,version=version+1 WHERE workspace_id=$1 AND id=$2 AND version=$19", item.WorkspaceID, item.ID, item.Title, item.Description, item.PipelineID, item.StageID, item.ContactID, item.CompanyID, item.AssignedUserID, item.ExpectedRevenue, item.Currency, item.ExpectedCloseDate, item.Outcome, item.LossReason, values, item.ClosedAt, item.UpdatedBy, item.UpdatedAt, version)
	if err != nil {
		return mapErr(err)
	}
	if x.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	item.Version = version + 1
	return nil
}
func (r *Repository) CreateHistory(ctx context.Context, item domain.History) error {
	_, err := r.q(ctx).Exec(ctx, "INSERT INTO crm_lifecycle_history(id,workspace_id,lead_id,opportunity_id,event_type,from_stage_id,to_stage_id,actor_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)", item.ID, item.WorkspaceID, item.LeadID, item.OpportunityID, item.EventType, item.FromStageID, item.ToStageID, item.ActorID, item.CreatedAt)
	return mapErr(err)
}
func (r *Repository) ListHistory(ctx context.Context, w uuid.UUID, leadID, opportunityID *uuid.UUID) ([]domain.History, error) {
	query, args := "SELECT id,workspace_id,lead_id,opportunity_id,event_type,from_stage_id,to_stage_id,actor_id,created_at FROM crm_lifecycle_history WHERE workspace_id=$1", []any{w}
	if leadID != nil {
		query += " AND lead_id=$2"
		args = append(args, *leadID)
	} else if opportunityID != nil {
		query += " AND opportunity_id=$2"
		args = append(args, *opportunityID)
	} else {
		return nil, fmt.Errorf("history target is required")
	}
	rows, err := r.q(ctx).Query(ctx, query+" ORDER BY created_at,id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.History{}
	for rows.Next() {
		var item domain.History
		if err = rows.Scan(&item.ID, &item.WorkspaceID, &item.LeadID, &item.OpportunityID, &item.EventType, &item.FromStageID, &item.ToStageID, &item.ActorID, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

 // Supported built-in columns are fixed literals; keys for workspace custom
 // fields are always bound parameters. Never interpolate saved-view input into SQL.
var viewColumns = map[string]map[string]string{
    "crm.lead": {
        "title":"title","description":"description","contact_name":"contact_name","company_name":"company_name",
        "email":"email","phone":"phone","source":"source","status":"status",
        "assigned_user_id":"assigned_user_id","contact_id":"contact_id","created_at":"created_at","updated_at":"updated_at",
    },
    "crm.opportunity": {
        "title":"title","description":"description","pipeline_id":"pipeline_id","stage_id":"stage_id",
        "contact_id":"contact_id","company_id":"company_id","assigned_user_id":"assigned_user_id",
        "expected_revenue":"expected_revenue","currency":"currency","expected_close_date":"expected_close_date",
        "outcome":"outcome","loss_reason":"loss_reason","created_at":"created_at","updated_at":"updated_at",
    },
}

func viewExpression(column, kind string) string {
    switch kind {
    case "integer", "decimal": return "("+column+")::numeric"
    case "boolean": return "("+column+")::boolean"
    default: return "("+column+")::text"
    }
}

func applyViewConditions(where *[]string, args *[]any, conditions []domain.ViewCondition, entity string) error {
    if len(conditions) > 20 { return fmt.Errorf("saved view contains too many filters") }
    for _, condition := range conditions {
        var expression string
        if condition.Custom {
            *args = append(*args, condition.Field)
            expression = "custom_values->>$"+fmt.Sprint(len(*args))
        } else {
            column := viewColumns[entity][condition.Field]
            if column == "" { return fmt.Errorf("saved-view filter field %q is not supported by CRM", condition.Field) }
            expression = column
        }
        // Compare typed numbers numerically (not lexicographically).
        expression = viewExpression(expression, condition.Type)
        appendValue := func(value any) string {
            *args = append(*args, fmt.Sprint(value))
            return "$"+fmt.Sprint(len(*args))
        }
        switch condition.Operator {
        case "eq", "neq":
            op := "="
            if condition.Operator == "neq" { op = "<>" }
            *where = append(*where, expression+" "+op+" "+appendValue(condition.Value))
        case "contains":
            if condition.Type != "string" && condition.Type != "text" {
                return fmt.Errorf("contains requires a text field")
            }
            *where = append(*where, expression+" ILIKE "+appendValue("%"+fmt.Sprint(condition.Value)+"%"))
        case "in":
            items := []any{}
            switch values := condition.Value.(type) {
            case []any: items = values
            case []string:
                for _, v := range values { items = append(items,v) }
            default: return fmt.Errorf("in filter must have a value list")
            }
            if len(items)==0 || len(items)>100 { return fmt.Errorf("in filter list must contain between 1 and 100 values") }
            placeholders:=make([]string,0,len(items))
            for _, item:=range items { placeholders=append(placeholders,appendValue(item)) }
            *where=append(*where,expression+" IN ("+strings.Join(placeholders,",")+")")
        case "is_empty":
            if condition.Type=="string" || condition.Type=="text" {
                *where=append(*where,"("+expression+" IS NULL OR "+expression+" = '')")
            } else { *where=append(*where,expression+" IS NULL") }
        default: return fmt.Errorf("unsupported view operator %q", condition.Operator)
        }
    }
    return nil
}

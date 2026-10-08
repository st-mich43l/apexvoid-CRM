package application

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	contactsapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/api"
	customdomain "github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
	"strings"
	"time"
)

type Service struct {
	r      domain.Repository
	tx     *database.TxManager
	custom interface {
		ValidateCustomValues(context.Context, uuid.UUID, string, map[string]any) error
		ListViews(context.Context, uuid.UUID, uuid.UUID, string) ([]customdomain.SavedView, error)
		EffectiveSchema(context.Context, uuid.UUID, string) (customdomain.EffectiveSchema, error)
	}
	contacts interface {
		contactsapi.ContactReader
		contactsapi.ContactCreator
	}
	workspace organizationapi.WorkspaceResolver
	access    organizationapi.WorkspaceAccess
	events    *event.Bus
}

func New(r domain.Repository, tx *database.TxManager) *Service { return &Service{r: r, tx: tx} }

type Dependencies struct {
	Repository   domain.Repository
	Transactions *database.TxManager
	CustomValues interface {
		ValidateCustomValues(context.Context, uuid.UUID, string, map[string]any) error
		ListViews(context.Context, uuid.UUID, uuid.UUID, string) ([]customdomain.SavedView, error)
		EffectiveSchema(context.Context, uuid.UUID, string) (customdomain.EffectiveSchema, error)
	}
	Contacts interface {
		contactsapi.ContactReader
		contactsapi.ContactCreator
	}
	Workspace organizationapi.WorkspaceResolver
	Access    organizationapi.WorkspaceAccess
	Events    *event.Bus
}

func NewWithDependencies(d Dependencies) *Service {
	return &Service{r: d.Repository, tx: d.Transactions, custom: d.CustomValues, contacts: d.Contacts, workspace: d.Workspace, access: d.Access, events: d.Events}
}
func (s *Service) Templates() []domain.Template { return domain.Templates() }
func (s *Service) List(ctx context.Context, w uuid.UUID, a bool) ([]domain.Pipeline, error) {
	return s.r.ListPipelines(ctx, w, a)
}
func (s *Service) Get(ctx context.Context, w, id uuid.UUID) (*domain.Pipeline, error) {
	return s.r.GetPipeline(ctx, w, id)
}
func (s *Service) Stages(ctx context.Context, w, p uuid.UUID, a bool) ([]domain.Stage, error) {
	return s.r.ListStages(ctx, w, p, a)
}
func (s *Service) Initialize(ctx context.Context, w, u uuid.UUID, key string) (domain.Pipeline, error) {
	for _, t := range domain.Templates() {
		if t.Key == key {
			return s.create(ctx, w, u, t, t.Name, "")
		}
	}
	return domain.Pipeline{}, fmt.Errorf("template not found")
}
func (s *Service) Create(ctx context.Context, w, u uuid.UUID, name, description, color string) (domain.Pipeline, error) {
	return s.create(ctx, w, u, domain.Template{Key: slug(name), Name: name, Description: description, OpenStages: []domain.TemplateStage{{Key: "new", Name: "New", Probability: 10}}}, name, color)
}
func (s *Service) create(ctx context.Context, w, u uuid.UUID, t domain.Template, name, color string) (out domain.Pipeline, err error) {
	err = s.with(ctx, func(c context.Context) error {
		if e := s.r.LockWorkspace(c, w); e != nil {
			return e
		}
		items, e := s.r.ListPipelines(c, w, false)
		if e != nil {
			return e
		}
		for _, x := range items {
			if strings.EqualFold(x.Name, name) {
				return domain.ErrConflict
			}
		}
		now := time.Now().UTC()
		if strings.TrimSpace(color) == "" {
			color = "#7c3aed"
		}
		out = domain.Pipeline{ID: uuid.New(), WorkspaceID: w, Name: name, Slug: slug(name), Description: t.Description, Color: color, Status: domain.PipelineActive, Default: len(items) == 0, DisplayOrder: len(items), CreatedBy: u, UpdatedBy: u, Version: 1, CreatedAt: now, UpdatedAt: now}
		if e = out.Validate(); e != nil {
			return e
		}
		st := []domain.Stage{}
		for i, x := range t.OpenStages {
			st = append(st, domain.Stage{ID: uuid.New(), WorkspaceID: w, PipelineID: out.ID, Key: x.Key, Name: x.Name, Description: x.Description, Color: x.Color, Category: domain.StageOpen, Position: i, Probability: x.Probability, Active: true, Version: 1, CreatedAt: now, UpdatedAt: now})
		}
		st = append(st, domain.Stage{ID: uuid.New(), WorkspaceID: w, PipelineID: out.ID, Key: "won", Name: "Won", Category: domain.StageWon, Position: len(st), Probability: 100, Active: true, Version: 1, CreatedAt: now, UpdatedAt: now}, domain.Stage{ID: uuid.New(), WorkspaceID: w, PipelineID: out.ID, Key: "lost", Name: "Lost", Category: domain.StageLost, Position: len(st) + 1, Probability: 0, Active: true, Version: 1, CreatedAt: now, UpdatedAt: now})
		if e = domain.ValidatePipelineStages(st); e != nil {
			return e
		}
		if e = s.r.CreatePipeline(c, &out); e != nil {
			return e
		}
		return s.r.CreateStages(c, st)
	})
	return
}
func (s *Service) SetDefault(ctx context.Context, w, id uuid.UUID) error {
	return s.with(ctx, func(c context.Context) error {
		if e := s.r.LockWorkspace(c, w); e != nil {
			return e
		}
		return s.r.SetDefault(c, w, id)
	})
}
func (s *Service) UpdatePipeline(ctx context.Context, w, actor, id uuid.UUID, version int, name, description, color string) (domain.Pipeline, error) {
	var out domain.Pipeline
	err := s.with(ctx, func(c context.Context) error {
		if err := s.r.LockWorkspace(c, w); err != nil {
			return err
		}
		current, err := s.r.GetPipeline(c, w, id)
		if err != nil {
			return err
		}
		if current.Version != version {
			return domain.ErrConflict
		}
		current.Name, current.Description, current.Color, current.UpdatedBy, current.UpdatedAt = name, description, color, actor, time.Now().UTC()
		if err = current.Validate(); err != nil {
			return err
		}
		if err = s.r.UpdatePipeline(c, current, version); err != nil {
			return err
		}
		out = *current
		return nil
	})
	return out, err
}
func (s *Service) SetPipelineArchived(ctx context.Context, w, id uuid.UUID, version int, archived bool) error {
	return s.with(ctx, func(c context.Context) error {
		if err := s.r.LockWorkspace(c, w); err != nil {
			return err
		}
		item, err := s.r.GetPipeline(c, w, id)
		if err != nil {
			return err
		}
		if item.Version != version {
			return domain.ErrConflict
		}
		if archived {
			if item.Default {
				return fmt.Errorf("set another default pipeline before archiving this pipeline")
			}
			open, err := s.r.CountOpenOpportunities(c, w, id, nil)
			if err != nil {
				return err
			}
			if open > 0 {
				return fmt.Errorf("reassign or close open opportunities before archiving this pipeline")
			}
			return s.r.SetPipelineStatus(c, w, id, domain.PipelineArchived, version)
		}
		if err := s.r.SetPipelineStatus(c, w, id, domain.PipelineActive, version); err != nil {
			return err
		}
		pipelines, err := s.r.ListPipelines(c, w, false)
		if err != nil {
			return err
		}
		for _, pipeline := range pipelines {
			if pipeline.Default {
				return nil
			}
		}
		return s.r.SetDefault(c, w, id)
	})
}
func (s *Service) ClonePipeline(ctx context.Context, w, actor, id uuid.UUID, name string) (domain.Pipeline, error) {
	if strings.TrimSpace(name) == "" {
		return domain.Pipeline{}, fmt.Errorf("pipeline name is required")
	}
	var sourceStages []domain.Stage
	var source domain.Pipeline
	err := s.with(ctx, func(c context.Context) error {
		var err error
		if err = s.r.LockWorkspace(c, w); err != nil {
			return err
		}
		if sourcePtr, e := s.r.GetPipeline(c, w, id); e != nil {
			return e
		} else {
			source = *sourcePtr
		}
		sourceStages, err = s.r.ListStages(c, w, id, false)
		return err
	})
	if err != nil {
		return domain.Pipeline{}, err
	}
	template := domain.Template{Key: slug(name), Name: name, Description: source.Description}
	for _, stage := range sourceStages {
		if stage.Category == domain.StageOpen {
			template.OpenStages = append(template.OpenStages, domain.TemplateStage{Key: stage.Key, Name: stage.Name, Description: stage.Description, Color: stage.Color, Probability: stage.Probability})
		}
	}
	return s.create(ctx, w, actor, template, name, source.Color)
}
func (s *Service) AddStage(ctx context.Context, w, p uuid.UUID, key, name string, prob int) (domain.Stage, error) {
	var out domain.Stage
	e := s.with(ctx, func(c context.Context) error {
		if e := s.r.LockWorkspace(c, w); e != nil {
			return e
		}
		st, e := s.r.ListStages(c, w, p, true)
		if e != nil {
			return e
		}
		if _, e = s.r.GetPipeline(c, w, p); e != nil {
			return e
		}
		// Renumber first: the repository moves active stages to a collision-free
		// temporary range before applying the final sequence.
		order := make([]uuid.UUID, 0, len(st)+1)
		for _, item := range st {
			if item.Category == domain.StageOpen {
				order = append(order, item.ID)
			}
		}
		out = domain.Stage{ID: uuid.New(), WorkspaceID: w, PipelineID: p, Key: slug(key), Name: name, Category: domain.StageOpen, Position: len(order), Probability: prob, Active: true, Version: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		if e = out.Validate(); e != nil {
			return e
		}
		if e = s.r.SetTemporaryStagePositions(c, w, p); e != nil {
			return e
		}
		if e = s.r.CreateStages(c, []domain.Stage{out}); e != nil {
			return e
		}
		order = append(order, out.ID)
		for _, item := range st {
			if item.Category != domain.StageOpen {
				order = append(order, item.ID)
			}
		}
		if e = s.r.SetStagePositions(c, w, p, order); e != nil {
			return e
		}
		// Positioning is also an optimistic update, so return the version callers
		// must use for their first edit rather than the pre-positioning version.
		out.Version++
		return nil
	})
	return out, e
}
func (s *Service) UpdateStage(ctx context.Context, w, p, id uuid.UUID, version int, name, description, color string, probability int) (domain.Stage, error) {
	var out domain.Stage
	err := s.with(ctx, func(c context.Context) error {
		if err := s.r.LockWorkspace(c, w); err != nil {
			return err
		}
		stages, err := s.r.ListStages(c, w, p, true)
		if err != nil {
			return err
		}
		for _, stage := range stages {
			if stage.ID == id {
				out = stage
				break
			}
		}
		if out.ID == uuid.Nil {
			return domain.ErrNotFound
		}
		if out.Version != version {
			return domain.ErrConflict
		}
		out.Name, out.Description, out.Color = name, description, color
		if out.Category == domain.StageOpen {
			out.Probability = probability
		}
		out.UpdatedAt = time.Now().UTC()
		if err = out.Validate(); err != nil {
			return err
		}
		return s.r.UpdateStage(c, &out, version)
	})
	return out, err
}
func (s *Service) SetStageArchived(ctx context.Context, w, p, id uuid.UUID, archived bool) error {
	return s.with(ctx, func(c context.Context) error {
		if err := s.r.LockWorkspace(c, w); err != nil {
			return err
		}
		stages, err := s.r.ListStages(c, w, p, true)
		if err != nil {
			return err
		}
		var stage *domain.Stage
		for i := range stages {
			if stages[i].ID == id {
				stage = &stages[i]
				break
			}
		}
		if stage == nil {
			return domain.ErrNotFound
		}
		if archived {
			open, err := s.r.CountOpenOpportunities(c, w, p, &id)
			if err != nil {
				return err
			}
			if open > 0 {
				return fmt.Errorf("reassign open opportunities before archiving this stage")
			}
			if stage.Category != domain.StageOpen {
				return fmt.Errorf("terminal stages cannot be archived")
			}
			stage.Active = false
			if err := domain.ValidatePipelineStages(stages); err != nil {
				return err
			}
			return s.r.SetStagesActive(c, w, p, []uuid.UUID{id}, false)
		}
		stage.Active = true
		maxPosition := -1
		for _, item := range stages {
			if item.ID != stage.ID && item.Active && item.Position > maxPosition {
				maxPosition = item.Position
			}
		}
		stage.Position = maxPosition + 1
		if err := domain.ValidatePipelineStages(stages); err != nil {
			return err
		}
		if err := s.r.SetTemporaryStagePositions(c, w, p); err != nil {
			return err
		}
		if err := s.r.SetStagesActive(c, w, p, []uuid.UUID{id}, true); err != nil {
			return err
		}
		order := make([]uuid.UUID, 0, len(stages))
		for _, item := range stages {
			if item.Active && item.Category == domain.StageOpen {
				order = append(order, item.ID)
			}
		}
		for _, item := range stages {
			if item.Active && item.Category != domain.StageOpen {
				order = append(order, item.ID)
			}
		}
		return s.r.SetStagePositions(c, w, p, order)
	})
}
func (s *Service) Reorder(ctx context.Context, w, p uuid.UUID, ids []uuid.UUID) error {
	return s.with(ctx, func(c context.Context) error {
		if e := s.r.LockWorkspace(c, w); e != nil {
			return e
		}
		st, e := s.r.ListStages(c, w, p, false)
		if e != nil {
			return e
		}
		if len(st) != len(ids) {
			return fmt.Errorf("stage order must include every active stage")
		}
		seen := map[uuid.UUID]bool{}
		allowed := map[uuid.UUID]bool{}
		for _, x := range st {
			allowed[x.ID] = true
		}
		for _, id := range ids {
			if seen[id] || !allowed[id] {
				return fmt.Errorf("stage order is invalid")
			}
			seen[id] = true
		}
		if e = s.r.SetTemporaryStagePositions(c, w, p); e != nil {
			return e
		}
		return s.r.SetStagePositions(c, w, p, ids)
	})
}
func (s *Service) with(c context.Context, f func(context.Context) error) error {
	if s.tx == nil {
		return f(c)
	}
	return s.tx.WithTransaction(c, f)
}
func slug(x string) string {
	x = strings.ToLower(strings.TrimSpace(x))
	x = strings.ReplaceAll(x, " ", "_")
	return strings.Trim(x, "_")
}

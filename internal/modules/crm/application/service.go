package application

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	contactsapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/api"
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
		return s.r.SetStagePositions(c, w, p, order)
	})
	return out, e
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

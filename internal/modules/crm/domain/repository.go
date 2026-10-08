package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	LockWorkspace(context.Context, uuid.UUID) error
	ListPipelines(context.Context, uuid.UUID, bool) ([]Pipeline, error)
	GetPipeline(context.Context, uuid.UUID, uuid.UUID) (*Pipeline, error)
	CreatePipeline(context.Context, *Pipeline) error
	UpdatePipeline(context.Context, *Pipeline, int) error
	SetPipelineStatus(context.Context, uuid.UUID, uuid.UUID, PipelineStatus, int) error
	SetDefault(context.Context, uuid.UUID, uuid.UUID) error
	ListStages(context.Context, uuid.UUID, uuid.UUID, bool) ([]Stage, error)
	CreateStages(context.Context, []Stage) error
	UpdateStage(context.Context, *Stage, int) error
	SetStagesActive(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID, bool) error
	CountOpenOpportunities(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID) (int, error)
	SetTemporaryStagePositions(context.Context, uuid.UUID, uuid.UUID) error
	SetStagePositions(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error
	GetLead(context.Context, uuid.UUID, uuid.UUID) (*Lead, error)
	GetLeadForUpdate(context.Context, uuid.UUID, uuid.UUID) (*Lead, error)
	ListLeads(context.Context, uuid.UUID, LeadFilter) ([]Lead, int, error)
	CreateLead(context.Context, *Lead) error
	UpdateLead(context.Context, *Lead, int) error
	GetOpportunity(context.Context, uuid.UUID, uuid.UUID) (*Opportunity, error)
	GetOpportunityForUpdate(context.Context, uuid.UUID, uuid.UUID) (*Opportunity, error)
	ListOpportunities(context.Context, uuid.UUID, OpportunityFilter) ([]Opportunity, int, error)
	CreateOpportunity(context.Context, *Opportunity) error
	UpdateOpportunity(context.Context, *Opportunity, int) error
	CreateHistory(context.Context, History) error
	ListHistory(context.Context, uuid.UUID, *uuid.UUID, *uuid.UUID) ([]History, error)
}

type ViewCondition struct {
	Field string
	Operator string
	Value any
	Custom bool
	Type string
}

type LeadFilter struct {
	Search      string
	ViewID      *uuid.UUID
	ViewUserID  uuid.UUID
	Conditions  []ViewCondition
	SortCustom  bool
	SortType    string
	Status      LeadStatus
	OwnerID     *uuid.UUID
	Page, Limit int
	Sort        string
	Desc        bool
}
type OpportunityFilter struct {
	Search                       string
	ViewID                       *uuid.UUID
	ViewUserID                   uuid.UUID
	Conditions                   []ViewCondition
	SortCustom                   bool
	SortType                     string
	PipelineID, StageID, OwnerID *uuid.UUID
	Outcome                      OpportunityOutcome
	Page, Limit                  int
	Sort                         string
	Desc                         bool
}

var (
	ErrNotFound = errText("crm resource not found")
	ErrConflict = errText("crm configuration was changed by another user")
)

type errText string

func (e errText) Error() string { return string(e) }

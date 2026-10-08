package domain

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type LeadStatus string

const (
	LeadNew          LeadStatus = "new"
	LeadContacted    LeadStatus = "contacted"
	LeadQualified    LeadStatus = "qualified"
	LeadConverted    LeadStatus = "converted"
	LeadDisqualified LeadStatus = "disqualified"
)

type OpportunityOutcome string

const (
	OpportunityOpen OpportunityOutcome = "open"
	OpportunityWon  OpportunityOutcome = "won"
	OpportunityLost OpportunityOutcome = "lost"
)

// Lead retains its history after conversion; it is never replaced by an opportunity.
type Lead struct {
	ID                     uuid.UUID      `json:"id"`
	WorkspaceID            uuid.UUID      `json:"workspace_id"`
	Title                  string         `json:"title"`
	Description            string         `json:"description"`
	ContactName            string         `json:"contact_name"`
	CompanyName            string         `json:"company_name"`
	Email                  string         `json:"email"`
	Phone                  string         `json:"phone"`
	Source                 string         `json:"source"`
	AssignedUserID         *uuid.UUID     `json:"assigned_user_id,omitempty"`
	ContactID              *uuid.UUID     `json:"contact_id,omitempty"`
	ConvertedOpportunityID *uuid.UUID     `json:"converted_opportunity_id,omitempty"`
	Status                 LeadStatus     `json:"status"`
	DisqualificationReason string         `json:"disqualification_reason,omitempty"`
	CustomValues           map[string]any `json:"custom_values"`
	CreatedBy              uuid.UUID      `json:"created_by"`
	UpdatedBy              uuid.UUID      `json:"updated_by"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	ConvertedAt            *time.Time     `json:"converted_at,omitempty"`
	Version                int            `json:"version"`
}

// Opportunity uses a canonical decimal string for money. PostgreSQL persists it as
// NUMERIC(20,2); clients never exchange binary floating point values for revenue.
type Opportunity struct {
	ID                uuid.UUID          `json:"id"`
	WorkspaceID       uuid.UUID          `json:"workspace_id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	PipelineID        uuid.UUID          `json:"pipeline_id"`
	StageID           uuid.UUID          `json:"stage_id"`
	ContactID         *uuid.UUID         `json:"contact_id,omitempty"`
	CompanyID         *uuid.UUID         `json:"company_id,omitempty"`
	AssignedUserID    *uuid.UUID         `json:"assigned_user_id,omitempty"`
	ExpectedRevenue   string             `json:"expected_revenue"`
	Currency          string             `json:"currency"`
	ExpectedCloseDate *time.Time         `json:"expected_close_date,omitempty"`
	Outcome           OpportunityOutcome `json:"outcome"`
	LossReason        string             `json:"loss_reason,omitempty"`
	CustomValues      map[string]any     `json:"custom_values"`
	OriginalLeadID    *uuid.UUID         `json:"original_lead_id,omitempty"`
	CreatedBy         uuid.UUID          `json:"created_by"`
	UpdatedBy         uuid.UUID          `json:"updated_by"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	ClosedAt          *time.Time         `json:"closed_at,omitempty"`
	Version           int                `json:"version"`
}

// History is durable audit data for lifecycle operations; events remain a
// post-commit integration notification rather than the audit source of truth.
type History struct {
	ID            uuid.UUID  `json:"id"`
	WorkspaceID   uuid.UUID  `json:"workspace_id"`
	LeadID        *uuid.UUID `json:"lead_id,omitempty"`
	OpportunityID *uuid.UUID `json:"opportunity_id,omitempty"`
	EventType     string     `json:"event_type"`
	FromStageID   *uuid.UUID `json:"from_stage_id,omitempty"`
	ToStageID     *uuid.UUID `json:"to_stage_id,omitempty"`
	ActorID       uuid.UUID  `json:"actor_id"`
	CreatedAt     time.Time  `json:"created_at"`
}

var decimalAmount = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]{1,2})?$`)

func (l Lead) Validate() error {
	if strings.TrimSpace(l.Title) == "" {
		return fmt.Errorf("lead title is required")
	}
	if l.Status != LeadNew && l.Status != LeadContacted && l.Status != LeadQualified && l.Status != LeadConverted && l.Status != LeadDisqualified {
		return fmt.Errorf("lead status is invalid")
	}
	if l.Status == LeadDisqualified && strings.TrimSpace(l.DisqualificationReason) == "" {
		return fmt.Errorf("disqualification reason is required")
	}
	if l.Status == LeadConverted && (l.ConvertedOpportunityID == nil || l.ConvertedAt == nil) {
		return fmt.Errorf("converted lead requires opportunity and timestamp")
	}
	return nil
}

func (l *Lead) Transition(next LeadStatus, reason string) error {
	allowed := map[LeadStatus][]LeadStatus{LeadNew: {LeadContacted, LeadDisqualified}, LeadContacted: {LeadQualified, LeadDisqualified}, LeadQualified: {LeadConverted, LeadDisqualified}, LeadDisqualified: {LeadNew}, LeadConverted: {}}
	for _, candidate := range allowed[l.Status] {
		if candidate == next {
			l.Status = next
			if next == LeadDisqualified {
				l.DisqualificationReason = strings.TrimSpace(reason)
			}
			if next == LeadNew {
				l.DisqualificationReason = ""
			}
			return l.Validate()
		}
	}
	return fmt.Errorf("lead transition from %s to %s is not allowed", l.Status, next)
}

// Convert is deliberately separate from Transition: conversion is the only
// lifecycle change that has mandatory reciprocal persistence state.
func (l *Lead) Convert(opportunityID uuid.UUID, convertedAt time.Time) error {
	if l.Status != LeadQualified {
		return fmt.Errorf("only qualified leads can be converted")
	}
	if opportunityID == uuid.Nil || convertedAt.IsZero() {
		return fmt.Errorf("converted lead requires opportunity and timestamp")
	}
	l.Status = LeadConverted
	l.ConvertedOpportunityID = &opportunityID
	l.ConvertedAt = &convertedAt
	return l.Validate()
}

func (o Opportunity) Validate() error {
	if strings.TrimSpace(o.Title) == "" {
		return fmt.Errorf("opportunity title is required")
	}
	if o.PipelineID == uuid.Nil || o.StageID == uuid.Nil {
		return fmt.Errorf("pipeline and stage are required")
	}
	if !decimalAmount.MatchString(o.ExpectedRevenue) {
		return fmt.Errorf("expected revenue must be a non-negative decimal with at most two places")
	}
	amount, ok := new(big.Rat).SetString(o.ExpectedRevenue)
	if !ok || amount.Sign() < 0 {
		return fmt.Errorf("expected revenue is invalid")
	}
	if _, ok := supportedCurrencies[strings.ToUpper(o.Currency)]; !ok {
		return fmt.Errorf("currency is unsupported")
	}
	if o.Outcome != OpportunityOpen && o.Outcome != OpportunityWon && o.Outcome != OpportunityLost {
		return fmt.Errorf("opportunity outcome is invalid")
	}
	if o.Outcome == OpportunityLost && strings.TrimSpace(o.LossReason) == "" {
		return fmt.Errorf("loss reason is required")
	}
	return nil
}

var supportedCurrencies = map[string]struct{}{"USD": {}, "EUR": {}, "GBP": {}, "JPY": {}, "AUD": {}, "CAD": {}, "CHF": {}, "CNY": {}, "THB": {}, "SGD": {}}

type PipelineStatus string

const (
	PipelineActive   PipelineStatus = "active"
	PipelineArchived PipelineStatus = "archived"
)

type StageCategory string

const (
	StageOpen StageCategory = "open"
	StageWon  StageCategory = "won"
	StageLost StageCategory = "lost"
)

type Pipeline struct {
	ID           uuid.UUID      `json:"id"`
	WorkspaceID  uuid.UUID      `json:"workspace_id"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	Description  string         `json:"description"`
	Color        string         `json:"color"`
	Status       PipelineStatus `json:"status"`
	Default      bool           `json:"default"`
	DisplayOrder int            `json:"display_order"`
	Version      int            `json:"version"`
	CreatedBy    uuid.UUID      `json:"created_by"`
	UpdatedBy    uuid.UUID      `json:"updated_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}
type Stage struct {
	ID          uuid.UUID     `json:"id"`
	WorkspaceID uuid.UUID     `json:"workspace_id"`
	PipelineID  uuid.UUID     `json:"pipeline_id"`
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Color       string        `json:"color"`
	Category    StageCategory `json:"category"`
	Position    int           `json:"position"`
	Probability int           `json:"probability"`
	Version     int           `json:"version"`
	Active      bool          `json:"active"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
type Template struct {
	Key, Name, Description string
	OpenStages             []TemplateStage
}
type TemplateStage struct {
	Key, Name, Description, Color string
	Probability                   int
}

func (p Pipeline) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if strings.TrimSpace(p.Slug) == "" {
		return fmt.Errorf("pipeline slug is required")
	}
	if p.Status != PipelineActive && p.Status != PipelineArchived {
		return fmt.Errorf("pipeline status is invalid")
	}
	if p.Status == PipelineArchived && p.Default {
		return fmt.Errorf("archived pipeline cannot be default")
	}
	return nil
}
func (s Stage) Validate() error {
	if strings.TrimSpace(s.Key) == "" || strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("stage key and name are required")
	}
	if s.Category != StageOpen && s.Category != StageWon && s.Category != StageLost {
		return fmt.Errorf("stage category is invalid")
	}
	if s.Position < 0 || s.Probability < 0 || s.Probability > 100 {
		return fmt.Errorf("stage position or probability is invalid")
	}
	if s.Category == StageWon && s.Probability != 100 {
		return fmt.Errorf("won stage probability must be 100")
	}
	if s.Category == StageLost && s.Probability != 0 {
		return fmt.Errorf("lost stage probability must be 0")
	}
	return nil
}
func ValidatePipelineStages(stages []Stage) error {
	open, won, lost := 0, 0, 0
	keys := map[string]bool{}
	positions := map[int]bool{}
	var workspaceID, pipelineID uuid.UUID
	for _, s := range stages {
		if err := s.Validate(); err != nil {
			return err
		}
		if keys[s.Key] {
			return fmt.Errorf("stage keys must be unique")
		}
		keys[s.Key] = true
		if !s.Active {
			continue
		}
		if workspaceID == uuid.Nil {
			workspaceID, pipelineID = s.WorkspaceID, s.PipelineID
		}
		if s.WorkspaceID != workspaceID || s.PipelineID != pipelineID {
			return fmt.Errorf("stages must share workspace and pipeline")
		}
		if positions[s.Position] {
			return fmt.Errorf("active stage positions must be unique")
		}
		positions[s.Position] = true
		switch s.Category {
		case StageOpen:
			open++
		case StageWon:
			won++
		case StageLost:
			lost++
		}
	}
	if open == 0 || won != 1 || lost != 1 {
		return fmt.Errorf("active pipeline requires open, exactly one won, and exactly one lost stage")
	}
	return nil
}

func Templates() []Template {
	return []Template{{Key: "standard_b2b", Name: "Standard B2B Sales", Description: "A practical B2B journey from qualification to negotiation.", OpenStages: []TemplateStage{{"new", "New", "Initial inbound or outbound interest.", "#7c3aed", 10}, {"qualified", "Qualified", "Confirmed fit and next step.", "#2563eb", 25}, {"proposal", "Proposal", "Commercial proposal shared.", "#d97706", 60}, {"negotiation", "Negotiation", "Terms and stakeholders aligned.", "#ea580c", 80}}}, {Key: "service_sales", Name: "Service Sales", Description: "A consultative service-sales journey.", OpenStages: []TemplateStage{{"inquiry", "Inquiry", "New service request.", "#7c3aed", 10}, {"consultation", "Consultation", "Discovery and consultation underway.", "#2563eb", 30}, {"quotation", "Quotation", "Quote prepared and shared.", "#d97706", 60}, {"approval", "Approval", "Awaiting final approval.", "#ea580c", 85}}}, {Key: "blank", Name: "Blank Pipeline", Description: "One starting stage plus final outcomes.", OpenStages: []TemplateStage{{"new", "New", "First stage for this pipeline.", "#7c3aed", 10}}}}
}

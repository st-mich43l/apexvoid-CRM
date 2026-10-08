package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
	ID, WorkspaceID                uuid.UUID
	Name, Slug, Description, Color string
	Status                         PipelineStatus
	Default                        bool
	DisplayOrder, Version          int
	CreatedBy, UpdatedBy           uuid.UUID
	CreatedAt, UpdatedAt           time.Time
}
type Stage struct {
	ID, WorkspaceID, PipelineID    uuid.UUID
	Key, Name, Description, Color  string
	Category                       StageCategory
	Position, Probability, Version int
	Active                         bool
	CreatedAt, UpdatedAt           time.Time
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

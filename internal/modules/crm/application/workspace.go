package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	contactsdomain "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type ConvertInput struct {
	PipelineID      uuid.UUID
	StageID         uuid.UUID
	ContactID       *uuid.UUID
	CreateContact   bool
	ExpectedRevenue string
	Currency        string
}

type LeadPatch struct {
	Title, Description, ContactName, CompanyName, Email, Phone, Source *string
	AssignedUserID, ContactID                                          *uuid.UUID
	SetAssignedUser, SetContact                                        bool
	CustomValues                                                       map[string]any
	SetCustomValues                                                    bool
}

type OpportunityPatch struct {
	Title, Description, ExpectedRevenue, Currency *string
	ExpectedCloseDate                             *time.Time
	AssignedUserID, ContactID, CompanyID          *uuid.UUID
	SetExpectedCloseDate, SetAssignedUser         bool
	SetContact, SetCompany                        bool
	CustomValues                                  map[string]any
	SetCustomValues                               bool
}

func (s *Service) ListLeads(ctx context.Context, workspaceID uuid.UUID, filter domain.LeadFilter) ([]domain.Lead, int, error) {
	if filter.ViewID != nil {
		conditions, sort, desc, custom, kind, err := s.resolveView(ctx, workspaceID, filter.ViewUserID, *filter.ViewID, "crm.lead")
		if err != nil {
			return nil, 0, err
		}
		filter.Conditions, filter.Sort, filter.Desc, filter.SortCustom, filter.SortType = conditions, sort, desc, custom, kind
	}
	return s.r.ListLeads(ctx, workspaceID, filter)
}
func (s *Service) GetLead(ctx context.Context, workspaceID, id uuid.UUID) (*domain.Lead, error) {
	return s.r.GetLead(ctx, workspaceID, id)
}
func (s *Service) ListOpportunities(ctx context.Context, workspaceID uuid.UUID, filter domain.OpportunityFilter) ([]domain.Opportunity, int, error) {
	if filter.ViewID != nil {
		conditions, sort, desc, custom, kind, err := s.resolveView(ctx, workspaceID, filter.ViewUserID, *filter.ViewID, "crm.opportunity")
		if err != nil {
			return nil, 0, err
		}
		filter.Conditions, filter.Sort, filter.Desc, filter.SortCustom, filter.SortType = conditions, sort, desc, custom, kind
	}
	return s.r.ListOpportunities(ctx, workspaceID, filter)
}

func (s *Service) OpportunityBoard(ctx context.Context, workspaceID, pipelineID uuid.UUID, filter domain.OpportunityBoardFilter) (domain.OpportunityBoard, error) {
	if filter.ViewID != nil {
		conditions, _, _, _, _, err := s.resolveView(ctx, workspaceID, filter.ViewUserID, *filter.ViewID, "crm.opportunity")
		if err != nil {
			return domain.OpportunityBoard{}, err
		}
		filter.Conditions = conditions
	}
	pipeline, err := s.r.GetPipeline(ctx, workspaceID, pipelineID)
	if err != nil {
		return domain.OpportunityBoard{}, err
	}
	if pipeline.Status != domain.PipelineActive {
		return domain.OpportunityBoard{}, fmt.Errorf("archived pipelines cannot be opened as a board")
	}
	if filter.Outcome != "" && filter.Outcome != domain.OpportunityOpen && filter.Outcome != domain.OpportunityWon && filter.Outcome != domain.OpportunityLost {
		return domain.OpportunityBoard{}, fmt.Errorf("unsupported opportunity outcome %q", filter.Outcome)
	}
	stages, err := s.r.ListStages(ctx, workspaceID, pipelineID, false)
	if err != nil {
		return domain.OpportunityBoard{}, err
	}
	category := domain.StageOpen
	if filter.Outcome == domain.OpportunityWon {
		category = domain.StageWon
	} else if filter.Outcome == domain.OpportunityLost {
		category = domain.StageLost
	}
	boardStages := make([]domain.Stage, 0, len(stages))
	validStage := false
	for _, stage := range stages {
		if stage.Active && stage.Category == category {
			boardStages = append(boardStages, stage)
			if filter.StageID != nil && stage.ID == *filter.StageID {
				validStage = true
			}
		}
	}
	if filter.StageID != nil && !validStage {
		return domain.OpportunityBoard{}, fmt.Errorf("stage does not belong to the selected board")
	}
	items, err := s.r.GetOpportunityBoard(ctx, workspaceID, pipelineID, boardStages, filter)
	if err != nil {
		return domain.OpportunityBoard{}, err
	}
	return domain.OpportunityBoard{Pipeline: *pipeline, Stages: items}, nil
}
func (s *Service) GetOpportunity(ctx context.Context, workspaceID, id uuid.UUID) (*domain.Opportunity, error) {
	return s.r.GetOpportunity(ctx, workspaceID, id)
}
func (s *Service) LeadHistory(ctx context.Context, workspaceID, id uuid.UUID) ([]domain.History, error) {
	return s.r.ListHistory(ctx, workspaceID, &id, nil)
}
func (s *Service) OpportunityHistory(ctx context.Context, workspaceID, id uuid.UUID) ([]domain.History, error) {
	return s.r.ListHistory(ctx, workspaceID, nil, &id)
}

func (s *Service) CreateLead(ctx context.Context, item domain.Lead) (domain.Lead, error) {
	item.ID = uuid.New()
	item.Title = strings.TrimSpace(item.Title)
	item.Status = domain.LeadNew
	item.Version = 1
	item.CustomValues = values(item.CustomValues)
	if err := s.validateLead(ctx, &item); err != nil {
		return domain.Lead{}, err
	}
	now := time.Now().UTC()
	item.CreatedAt, item.UpdatedAt = now, now
	if err := s.with(ctx, func(tx context.Context) error { return s.r.CreateLead(tx, &item) }); err != nil {
		return domain.Lead{}, err
	}
	s.publish(ctx, "crm.lead.created", item.WorkspaceID, item.ID)
	return item, nil
}

func (s *Service) UpdateLead(ctx context.Context, item domain.Lead, expectedVersion int) (domain.Lead, error) {
	current, err := s.r.GetLead(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		return domain.Lead{}, err
	}
	if current.Status == domain.LeadConverted {
		return domain.Lead{}, fmt.Errorf("converted leads cannot be edited")
	}
	item.Status, item.ConvertedAt, item.ConvertedOpportunityID = current.Status, current.ConvertedAt, current.ConvertedOpportunityID
	item.DisqualificationReason = current.DisqualificationReason
	item.CreatedAt, item.CreatedBy, item.Version = current.CreatedAt, current.CreatedBy, current.Version
	item.CustomValues = values(item.CustomValues)
	item.Title = strings.TrimSpace(item.Title)
	item.UpdatedAt = time.Now().UTC()
	if err := s.validateLead(ctx, &item); err != nil {
		return domain.Lead{}, err
	}
	if err := s.with(ctx, func(tx context.Context) error { return s.r.UpdateLead(tx, &item, expectedVersion) }); err != nil {
		return domain.Lead{}, err
	}
	s.publish(ctx, "crm.lead.updated", item.WorkspaceID, item.ID)
	return item, nil
}

func (s *Service) PatchLead(ctx context.Context, workspaceID, actor, id uuid.UUID, expectedVersion int, patch LeadPatch) (domain.Lead, error) {
	var item *domain.Lead
	err := s.with(ctx, func(tx context.Context) error {
		var err error
		item, err = s.r.GetLeadForUpdate(tx, workspaceID, id)
		if err != nil {
			return err
		}
		if item.Status == domain.LeadConverted {
			return fmt.Errorf("converted leads cannot be edited")
		}
		if item.Version != expectedVersion {
			return domain.ErrConflict
		}
		if patch.Title != nil {
			item.Title = strings.TrimSpace(*patch.Title)
		}
		if patch.Description != nil {
			item.Description = *patch.Description
		}
		if patch.ContactName != nil {
			item.ContactName = *patch.ContactName
		}
		if patch.CompanyName != nil {
			item.CompanyName = *patch.CompanyName
		}
		if patch.Email != nil {
			item.Email = *patch.Email
		}
		if patch.Phone != nil {
			item.Phone = *patch.Phone
		}
		if patch.Source != nil {
			item.Source = *patch.Source
		}
		if patch.SetAssignedUser {
			item.AssignedUserID = patch.AssignedUserID
		}
		if patch.SetContact {
			item.ContactID = patch.ContactID
		}
		if patch.SetCustomValues {
			item.CustomValues = mergeCustomValues(item.CustomValues, patch.CustomValues)
		}
		item.UpdatedBy, item.UpdatedAt = actor, time.Now().UTC()
		if err = s.validateLead(tx, item); err != nil {
			return err
		}
		return s.r.UpdateLead(tx, item, expectedVersion)
	})
	if err != nil {
		return domain.Lead{}, err
	}
	s.publish(ctx, "crm.lead.updated", workspaceID, id)
	return *item, nil
}

func (s *Service) TransitionLead(ctx context.Context, workspaceID, actor, id uuid.UUID, next domain.LeadStatus, reason string, version int) (domain.Lead, error) {
	var item *domain.Lead
	err := s.with(ctx, func(tx context.Context) error {
		var err error
		item, err = s.r.GetLeadForUpdate(tx, workspaceID, id)
		if err != nil {
			return err
		}
		if err = item.Transition(next, reason); err != nil {
			return err
		}
		item.UpdatedBy, item.UpdatedAt = actor, time.Now().UTC()
		if err = s.r.UpdateLead(tx, item, version); err != nil {
			return err
		}
		return s.recordHistory(tx, workspaceID, actor, &id, nil, "lead."+string(next), nil, nil)
	})
	if err != nil {
		return domain.Lead{}, err
	}
	name := "crm.lead.qualified"
	if next == domain.LeadDisqualified {
		name = "crm.lead.disqualified"
	}
	s.publish(ctx, name, workspaceID, id)
	return *item, nil
}

func (s *Service) CreateOpportunity(ctx context.Context, item domain.Opportunity) (domain.Opportunity, error) {
	item.ID = uuid.New()
	item.Outcome = domain.OpportunityOpen
	item.Version = 1
	item.ExpectedRevenue = strings.TrimSpace(item.ExpectedRevenue)
	item.Currency = strings.ToUpper(strings.TrimSpace(item.Currency))
	item.CustomValues = values(item.CustomValues)
	if err := s.validateOpportunity(ctx, &item); err != nil {
		return domain.Opportunity{}, err
	}
	now := time.Now().UTC()
	item.CreatedAt, item.UpdatedAt = now, now
	if err := s.with(ctx, func(tx context.Context) error { return s.r.CreateOpportunity(tx, &item) }); err != nil {
		return domain.Opportunity{}, err
	}
	s.publish(ctx, "crm.opportunity.created", item.WorkspaceID, item.ID)
	return item, nil
}

func (s *Service) UpdateOpportunity(ctx context.Context, item domain.Opportunity, expectedVersion int) (domain.Opportunity, error) {
	current, err := s.r.GetOpportunity(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		return domain.Opportunity{}, err
	}
	item.Outcome, item.ClosedAt, item.OriginalLeadID = current.Outcome, current.ClosedAt, current.OriginalLeadID
	item.LossReason = current.LossReason
	item.CreatedAt, item.CreatedBy, item.Version = current.CreatedAt, current.CreatedBy, current.Version
	item.ExpectedRevenue = strings.TrimSpace(item.ExpectedRevenue)
	item.Currency = strings.ToUpper(strings.TrimSpace(item.Currency))
	item.CustomValues = values(item.CustomValues)
	item.UpdatedAt = time.Now().UTC()
	if err := s.validateOpportunity(ctx, &item); err != nil {
		return domain.Opportunity{}, err
	}
	if err := s.with(ctx, func(tx context.Context) error { return s.r.UpdateOpportunity(tx, &item, expectedVersion) }); err != nil {
		return domain.Opportunity{}, err
	}
	s.publish(ctx, "crm.opportunity.updated", item.WorkspaceID, item.ID)
	return item, nil
}

func (s *Service) PatchOpportunity(ctx context.Context, workspaceID, actor, id uuid.UUID, expectedVersion int, patch OpportunityPatch) (domain.Opportunity, error) {
	var item *domain.Opportunity
	err := s.with(ctx, func(tx context.Context) error {
		var err error
		item, err = s.r.GetOpportunityForUpdate(tx, workspaceID, id)
		if err != nil {
			return err
		}
		if item.Version != expectedVersion {
			return domain.ErrConflict
		}
		if patch.Title != nil {
			item.Title = strings.TrimSpace(*patch.Title)
		}
		if patch.Description != nil {
			item.Description = *patch.Description
		}
		if patch.ExpectedRevenue != nil {
			item.ExpectedRevenue = strings.TrimSpace(*patch.ExpectedRevenue)
		}
		if patch.Currency != nil {
			item.Currency = strings.ToUpper(strings.TrimSpace(*patch.Currency))
		}
		if patch.SetExpectedCloseDate {
			item.ExpectedCloseDate = patch.ExpectedCloseDate
		}
		if patch.SetAssignedUser {
			item.AssignedUserID = patch.AssignedUserID
		}
		if patch.SetContact {
			item.ContactID = patch.ContactID
		}
		if patch.SetCompany {
			item.CompanyID = patch.CompanyID
		}
		if patch.SetCustomValues {
			item.CustomValues = mergeCustomValues(item.CustomValues, patch.CustomValues)
		}
		item.UpdatedBy, item.UpdatedAt = actor, time.Now().UTC()
		if err = s.validateOpportunity(tx, item); err != nil {
			return err
		}
		return s.r.UpdateOpportunity(tx, item, expectedVersion)
	})
	if err != nil {
		return domain.Opportunity{}, err
	}
	s.publish(ctx, "crm.opportunity.updated", workspaceID, id)
	return *item, nil
}

func (s *Service) MoveOpportunity(ctx context.Context, workspaceID, actor, id, pipelineID, stageID uuid.UUID, version int) (domain.Opportunity, error) {
	var item *domain.Opportunity
	err := s.with(ctx, func(tx context.Context) error {
		var err error
		item, err = s.r.GetOpportunityForUpdate(tx, workspaceID, id)
		if err != nil {
			return err
		}
		if item.Outcome != domain.OpportunityOpen {
			return fmt.Errorf("closed opportunity cannot move stages")
		}
		from := item.StageID
		item.PipelineID, item.StageID, item.UpdatedBy, item.UpdatedAt = pipelineID, stageID, actor, time.Now().UTC()
		if err = s.validateOpportunity(tx, item); err != nil {
			return err
		}
		if err = s.r.UpdateOpportunity(tx, item, version); err != nil {
			return err
		}
		return s.recordHistory(tx, workspaceID, actor, nil, &id, "opportunity.stage_changed", &from, &stageID)
	})
	if err != nil {
		return domain.Opportunity{}, err
	}
	s.publish(ctx, "crm.opportunity.stage_changed", workspaceID, id)
	return *item, nil
}

func (s *Service) CloseOpportunity(ctx context.Context, workspaceID, actor, id uuid.UUID, won bool, reason string, version int) (domain.Opportunity, error) {
	var item *domain.Opportunity
	err := s.with(ctx, func(tx context.Context) error {
		var err error
		item, err = s.r.GetOpportunityForUpdate(tx, workspaceID, id)
		if err != nil {
			return err
		}
		if item.Outcome != domain.OpportunityOpen {
			return fmt.Errorf("opportunity is already closed")
		}
		stages, err := s.r.ListStages(tx, workspaceID, item.PipelineID, false)
		if err != nil {
			return err
		}
		target := domain.StageWon
		if !won {
			target = domain.StageLost
		}
		from := item.StageID
		for _, stage := range stages {
			if stage.Category == target {
				item.StageID = stage.ID
				break
			}
		}
		if item.StageID == uuid.Nil {
			return fmt.Errorf("pipeline has no active %s stage", target)
		}
		now := time.Now().UTC()
		if won {
			item.Outcome = domain.OpportunityWon
			item.LossReason = ""
		} else {
			item.Outcome = domain.OpportunityLost
			item.LossReason = strings.TrimSpace(reason)
		}
		item.ClosedAt = &now
		item.UpdatedBy, item.UpdatedAt = actor, now
		if err = s.validateOpportunity(tx, item); err != nil {
			return err
		}
		if err = s.r.UpdateOpportunity(tx, item, version); err != nil {
			return err
		}
		event := "opportunity.won"
		if !won {
			event = "opportunity.lost"
		}
		return s.recordHistory(tx, workspaceID, actor, nil, &id, event, &from, &item.StageID)
	})
	if err != nil {
		return domain.Opportunity{}, err
	}
	if won {
		s.publish(ctx, "crm.opportunity.won", workspaceID, id)
	} else {
		s.publish(ctx, "crm.opportunity.lost", workspaceID, id)
	}
	return *item, nil
}

func (s *Service) ReopenOpportunity(ctx context.Context, workspaceID, actor, id, stageID uuid.UUID, version int) (domain.Opportunity, error) {
	var item *domain.Opportunity
	err := s.with(ctx, func(tx context.Context) error {
		var err error
		item, err = s.r.GetOpportunityForUpdate(tx, workspaceID, id)
		if err != nil {
			return err
		}
		if item.Outcome == domain.OpportunityOpen {
			return fmt.Errorf("opportunity is already open")
		}
		from := item.StageID
		item.StageID = stageID
		item.Outcome = domain.OpportunityOpen
		item.LossReason = ""
		item.ClosedAt = nil
		item.UpdatedBy, item.UpdatedAt = actor, time.Now().UTC()
		if err = s.validateOpportunity(tx, item); err != nil {
			return err
		}
		if err = s.r.UpdateOpportunity(tx, item, version); err != nil {
			return err
		}
		return s.recordHistory(tx, workspaceID, actor, nil, &id, "opportunity.reopened", &from, &stageID)
	})
	if err != nil {
		return domain.Opportunity{}, err
	}
	s.publish(ctx, "crm.opportunity.reopened", workspaceID, id)
	return *item, nil
}

func (s *Service) ConvertLead(ctx context.Context, workspaceID, actor, leadID uuid.UUID, input ConvertInput, version int) (domain.Opportunity, error) {
	var opportunity domain.Opportunity
	err := s.with(ctx, func(tx context.Context) error {
		lead, err := s.r.GetLeadForUpdate(tx, workspaceID, leadID)
		if err != nil {
			return err
		}
		if lead.Status == domain.LeadConverted {
			return domain.ErrConflict
		}
		if lead.Status != domain.LeadQualified {
			return fmt.Errorf("only qualified leads can be converted")
		}
		pipeline, err := s.r.GetPipeline(tx, workspaceID, input.PipelineID)
		if err != nil {
			return err
		}
		if pipeline.Status != domain.PipelineActive {
			return fmt.Errorf("destination pipeline is archived")
		}
		stages, err := s.r.ListStages(tx, workspaceID, input.PipelineID, false)
		if err != nil {
			return err
		}
		var stage *domain.Stage
		for i := range stages {
			if stages[i].ID == input.StageID {
				stage = &stages[i]
				break
			}
		}
		if stage == nil || stage.Category != domain.StageOpen {
			return fmt.Errorf("destination stage must be an active open stage")
		}
		contactID := input.ContactID
		if contactID != nil {
			if s.contacts == nil {
				return fmt.Errorf("contacts integration is unavailable")
			}
			if s.access != nil {
				allowed, permissionErr := s.access.CanInWorkspace(tx, actor, workspaceID, "contacts.contact.read")
				if permissionErr != nil {
					return permissionErr
				}
				if !allowed {
					return fmt.Errorf("contact read permission is required")
				}
			}
			if _, err = s.contacts.GetByID(tx, workspaceID, *contactID); err != nil {
				return err
			}
		} else if input.CreateContact {
			if s.contacts == nil {
				return fmt.Errorf("contacts integration is unavailable")
			}
			if s.access != nil {
				ok, e := s.access.CanInWorkspace(tx, actor, workspaceID, "contacts.contact.create")
				if e != nil {
					return e
				}
				if !ok {
					return fmt.Errorf("contact creation permission is required")
				}
			}
			contact, e := s.contacts.CreateContact(tx, contactsdomain.Contact{WorkspaceID: workspaceID, Kind: contactsdomain.KindPerson, DisplayName: lead.ContactName, Email: lead.Email, Phone: lead.Phone, Description: lead.Description, CreatedBy: actor, UpdatedBy: actor})
			if e != nil {
				return e
			}
			contactID = &contact.ID
		} else {
			return fmt.Errorf("select an existing contact or request explicit contact creation")
		}
		now := time.Now().UTC()
		opportunity = domain.Opportunity{ID: uuid.New(), WorkspaceID: workspaceID, Title: lead.Title, Description: lead.Description, PipelineID: input.PipelineID, StageID: input.StageID, ContactID: contactID, AssignedUserID: lead.AssignedUserID, ExpectedRevenue: input.ExpectedRevenue, Currency: strings.ToUpper(input.Currency), Outcome: domain.OpportunityOpen, CustomValues: map[string]any{}, OriginalLeadID: &lead.ID, CreatedBy: actor, UpdatedBy: actor, CreatedAt: now, UpdatedAt: now, Version: 1}
		if err = s.validateOpportunity(tx, &opportunity); err != nil {
			return err
		}
		if err = s.r.CreateOpportunity(tx, &opportunity); err != nil {
			return err
		}
		if err = lead.Convert(opportunity.ID, now); err != nil {
			return err
		}
		lead.ContactID = contactID
		lead.UpdatedBy, lead.UpdatedAt = actor, now
		if err = s.r.UpdateLead(tx, lead, version); err != nil {
			return err
		}
		return s.recordHistory(tx, workspaceID, actor, &leadID, &opportunity.ID, "lead.converted", nil, &input.StageID)
	})
	if err != nil {
		return domain.Opportunity{}, err
	}
	s.publish(ctx, "crm.lead.converted", workspaceID, leadID)
	s.publish(ctx, "crm.opportunity.created", workspaceID, opportunity.ID)
	return opportunity, nil
}

func (s *Service) validateLead(ctx context.Context, item *domain.Lead) error {
	if item.AssignedUserID != nil && s.workspace != nil {
		if _, err := s.workspace.ResolveWorkspaceContext(ctx, *item.AssignedUserID, item.WorkspaceID); err != nil {
			return fmt.Errorf("assigned user is not an active workspace member")
		}
	}
	if item.ContactID != nil && s.contacts != nil {
		if _, err := s.contacts.GetByID(ctx, item.WorkspaceID, *item.ContactID); err != nil {
			return err
		}
	}
	if s.custom != nil {
		if err := s.custom.ValidateCustomValues(ctx, item.WorkspaceID, "crm.lead", item.CustomValues); err != nil {
			return err
		}
	}
	return item.Validate()
}
func (s *Service) validateOpportunity(ctx context.Context, item *domain.Opportunity) error {
	if item.AssignedUserID != nil && s.workspace != nil {
		if _, err := s.workspace.ResolveWorkspaceContext(ctx, *item.AssignedUserID, item.WorkspaceID); err != nil {
			return fmt.Errorf("assigned user is not an active workspace member")
		}
	}
	if item.ContactID != nil && s.contacts != nil {
		if _, err := s.contacts.GetByID(ctx, item.WorkspaceID, *item.ContactID); err != nil {
			return err
		}
	}
	if item.CompanyID != nil && s.contacts != nil {
		if _, err := s.contacts.GetByID(ctx, item.WorkspaceID, *item.CompanyID); err != nil {
			return err
		}
	}
	pipeline, err := s.r.GetPipeline(ctx, item.WorkspaceID, item.PipelineID)
	if err != nil {
		return err
	}
	if pipeline.Status != domain.PipelineActive {
		return fmt.Errorf("pipeline is archived")
	}
	stages, err := s.r.ListStages(ctx, item.WorkspaceID, item.PipelineID, false)
	if err != nil {
		return err
	}
	var stage *domain.Stage
	for i := range stages {
		if stages[i].ID == item.StageID {
			stage = &stages[i]
			break
		}
	}
	if stage == nil {
		return fmt.Errorf("stage does not belong to pipeline")
	}
	if item.Outcome == domain.OpportunityOpen && stage.Category != domain.StageOpen {
		return fmt.Errorf("open opportunity requires an open stage")
	}
	if item.Outcome == domain.OpportunityWon && stage.Category != domain.StageWon {
		return fmt.Errorf("won opportunity requires the won stage")
	}
	if item.Outcome == domain.OpportunityLost && stage.Category != domain.StageLost {
		return fmt.Errorf("lost opportunity requires the lost stage")
	}
	if s.custom != nil {
		if err = s.custom.ValidateCustomValues(ctx, item.WorkspaceID, "crm.opportunity", item.CustomValues); err != nil {
			return err
		}
	}
	return item.Validate()
}
func values(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	return input
}
func mergeCustomValues(current, changes map[string]any) map[string]any {
	merged := make(map[string]any, len(current)+len(changes))
	for key, value := range current {
		merged[key] = value
	}
	for key, value := range changes {
		if value == nil {
			delete(merged, key)
		} else {
			merged[key] = value
		}
	}
	return merged
}
func (s *Service) recordHistory(ctx context.Context, workspaceID, actor uuid.UUID, leadID, opportunityID *uuid.UUID, eventType string, fromStageID, toStageID *uuid.UUID) error {
	return s.r.CreateHistory(ctx, domain.History{ID: uuid.New(), WorkspaceID: workspaceID, LeadID: leadID, OpportunityID: opportunityID, EventType: eventType, FromStageID: fromStageID, ToStageID: toStageID, ActorID: actor, CreatedAt: time.Now().UTC()})
}
func (s *Service) publish(ctx context.Context, name string, workspaceID, id uuid.UUID) {
	if s.events == nil {
		return
	}
	database.AfterCommit(ctx, func(callback context.Context) {
		_ = event.Publish(s.events, callback, name, domain.Event{WorkspaceID: workspaceID, ResourceID: id})
	})
}

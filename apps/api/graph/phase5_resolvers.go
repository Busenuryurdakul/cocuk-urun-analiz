package graph

import (
	"errors"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph/model"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/agent"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
)

func mapPhase5Error(err error) error {
	switch {
	case errors.Is(err, agent.ErrForbidden):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, evidence.ErrForbidden):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, agent.ErrInvalidInput), errors.Is(err, evidence.ErrInvalidInput):
		return gqlError("INVALID_INPUT", err)
	case errors.Is(err, agent.ErrComplianceRejected):
		return gqlError("COMPLIANCE_REJECTED", err)
	case errors.Is(err, agent.ErrRunNotCancellable):
		return gqlError("RUN_NOT_CANCELLABLE", err)
	default:
		return mapPhase4Error(err)
	}
}

func toModelAnalysisRun(run *domain.AnalysisRun) *model.AnalysisRun {
	out := &model.AnalysisRun{
		ID:                      run.ID.Hex(),
		OrganizationID:          run.OrganizationID.Hex(),
		ProductID:               run.ProductID.Hex(),
		ClientRequestID:         run.ClientRequestID,
		Status:                  model.AnalysisRunStatus(run.Status),
		TraceID:                 run.TraceID,
		ConfigSnapshotID:        run.ConfigSnapshotID.Hex(),
		ToolRegistryVersion:     run.ToolRegistryVersion,
		ToolPolicyVersion:       run.ToolPolicyVersion,
		CompliancePolicyVersion: run.CompliancePolicyVersion,
		PlannerVersion:          run.PlannerVersion,
		IterationCount:          run.IterationCount,
		CreatedAt:               run.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:               run.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if run.MarketplaceImportRunID != nil {
		v := run.MarketplaceImportRunID.Hex()
		out.MarketplaceImportRunID = &v
	}
	if run.CurrentPhase != "" {
		out.CurrentPhase = &run.CurrentPhase
	}
	if run.TerminalError != "" {
		out.TerminalError = &run.TerminalError
	}
	if run.TerminalReason != "" {
		out.TerminalReason = &run.TerminalReason
	}
	if run.StartedAt != nil {
		v := run.StartedAt.UTC().Format(time.RFC3339)
		out.StartedAt = &v
	}
	if run.CompletedAt != nil {
		v := run.CompletedAt.UTC().Format(time.RFC3339)
		out.CompletedAt = &v
	}
	return out
}

func toModelAgentRunEvents(events []domain.AgentRunEvent) []*model.AgentRunEvent {
	out := make([]*model.AgentRunEvent, 0, len(events))
	for i := range events {
		ev := events[i]
		meta := make([]*model.EventMetadataEntry, 0, len(ev.Metadata))
		for k, v := range ev.Metadata {
			meta = append(meta, &model.EventMetadataEntry{Key: k, Value: v})
		}
		item := &model.AgentRunEvent{
			ID:             ev.ID.Hex(),
			OrganizationID: ev.OrganizationID.Hex(),
			AnalysisRunID:  ev.RunID.Hex(),
			Sequence:       int(ev.Sequence),
			Phase:          ev.Phase,
			Status:         model.AgentEventStatus(ev.Status),
			Metadata:       meta,
			TraceID:        ev.TraceID,
			Timestamp:      ev.Timestamp.UTC().Format(time.RFC3339),
		}
		if ev.ToolName != "" {
			item.ToolName = &ev.ToolName
		}
		out = append(out, item)
	}
	return out
}

func domainAnalysisStatus(status *model.AnalysisRunStatus) *domain.AnalysisRunStatus {
	if status == nil {
		return nil
	}
	s := domain.AnalysisRunStatus(*status)
	return &s
}

func toModelEvidence(ev *domain.Evidence) *model.Evidence {
	out := &model.Evidence{
		ID:             ev.ID.Hex(),
		OrganizationID: ev.OrganizationID.Hex(),
		AnalysisRunID:  ev.AnalysisRunID.Hex(),
		ProductID:      ev.ProductID.Hex(),
		Source:         ev.Source,
		SourceType:     ev.SourceType,
		Claim:          ev.Claim,
		Reliability:    ev.Reliability,
		Freshness:      ev.Freshness,
		RetrievedAt:    ev.RetrievedAt.UTC().Format(time.RFC3339),
		CreatedAt:      ev.CreatedAt.UTC().Format(time.RFC3339),
	}
	if ev.ClaimID != "" {
		v := ev.ClaimID
		out.ClaimID = &v
	}
	if ev.Snippet != "" {
		v := ev.Snippet
		out.Snippet = &v
	}
	if ev.Reference != "" {
		v := ev.Reference
		out.Reference = &v
	}
	return out
}

func toModelEvidenceList(items []domain.Evidence) []*model.Evidence {
	out := make([]*model.Evidence, 0, len(items))
	for i := range items {
		out = append(out, toModelEvidence(&items[i]))
	}
	return out
}

package graph

import (
	"errors"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/graph/model"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/agent"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/analysis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func mapPhase5Error(err error) error {
	switch {
	case errors.Is(err, agent.ErrForbidden):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, evidence.ErrForbidden), errors.Is(err, safety.ErrForbidden), errors.Is(err, analysis.ErrForbidden):
		return gqlError("FORBIDDEN", errForbidden)
	case errors.Is(err, agent.ErrInvalidInput), errors.Is(err, evidence.ErrInvalidInput), errors.Is(err, safety.ErrInvalidInput), errors.Is(err, analysis.ErrInvalidInput):
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

func toModelReviewInsights(items []domain.ReviewSignal) []*model.ReviewInsight {
	out := make([]*model.ReviewInsight, 0, len(items))
	for _, item := range items {
		out = append(out, &model.ReviewInsight{
			Topic:   item.Topic,
			Count:   item.Count,
			Summary: item.Summary,
			Kind:    string(item.Kind),
		})
	}
	return out
}

func toModelFinalResult(final *domain.FinalAnalysisResult) *model.FinalAnalysisResult {
	if final == nil {
		return nil
	}
	limitations := final.Limitations
	if limitations == nil {
		limitations = []string{}
	}
	flags := final.HallucinationFlags
	if flags == nil {
		flags = []string{}
	}
	return &model.FinalAnalysisResult{
		SchemaVersion:      final.SchemaVersion,
		Summary:            final.Summary,
		OverallRisk:        final.OverallRisk,
		Confidence:         final.Confidence,
		Decision:           string(final.Decision),
		Recommendation:     final.Recommendation,
		Limitations:        limitations,
		HallucinationFlags: flags,
		CreatedAt:          final.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func attachEvidenceSupportStatus(items []*model.Evidence, vals []domain.EvidenceClaimValidation) {
	byID := map[string]string{}
	for _, v := range vals {
		for _, id := range v.EvidenceIDs {
			byID[id.Hex()] = string(v.SupportStatus)
		}
	}
	for _, ev := range items {
		if s, ok := byID[ev.ID]; ok {
			ev.SupportStatus = &s
		}
	}
}

func parseOrgRunLimit(organizationID, analysisRunID string, limit *int) (primitive.ObjectID, primitive.ObjectID, int, error) {
	orgID, err := parseObjectID(organizationID)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, 0, gqlError("INVALID_INPUT", err)
	}
	runID, err := parseObjectID(analysisRunID)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, 0, gqlError("INVALID_INPUT", err)
	}
	lim := 50
	if limit != nil {
		lim = *limit
	}
	return orgID, runID, lim, nil
}

func toModelSafetyFindings(items []domain.SafetyFinding) []*model.SafetyFinding {
	out := make([]*model.SafetyFinding, 0, len(items))
	for i := range items {
		f := items[i]
		ids := make([]string, 0, len(f.EvidenceIDs))
		for _, id := range f.EvidenceIDs {
			ids = append(ids, id.Hex())
		}
		out = append(out, &model.SafetyFinding{
			ID:            f.ID.Hex(),
			AnalysisRunID: f.AnalysisRunID.Hex(),
			ProductID:     f.ProductID.Hex(),
			Type:          string(f.Type),
			Severity:      string(f.Severity),
			Confidence:    f.Confidence,
			EvidenceIds:   ids,
			Rationale:     f.Rationale,
			CreatedAt:     f.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func toModelRecallMatches(items []domain.RecallMatchRecord) []*model.RecallMatch {
	out := make([]*model.RecallMatch, 0, len(items))
	for i := range items {
		m := items[i]
		item := &model.RecallMatch{
			Source:         m.Source,
			SourceRecordID: m.SourceRecordID,
			Matched:        m.Matched,
			Confidence:     m.Confidence,
			Method:         m.Method,
			RequiresReview: m.RequiresReview,
		}
		if m.Reference != "" {
			v := m.Reference
			item.Reference = &v
		}
		out = append(out, item)
	}
	return out
}

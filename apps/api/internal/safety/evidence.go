package safety

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/evidence"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

func (s *Service) createRecallEvidence(ctx context.Context, input AnalyzeInput, rec recall.Record, match recall.MatchResult) (*domain.Evidence, error) {
	sourceType := "OFFICIAL_RECALL"
	if rec.Source == recall.SourceGUBIS {
		sourceType = "OFFICIAL_SAFETY_NOTICE"
	}
	freshness := 0.8
	if !rec.PublishedAt.IsZero() && time.Since(rec.PublishedAt) > 365*24*time.Hour {
		freshness = 0.5
	}
	return s.Evidences.CreateEvidence(ctx, evidence.CreateEvidenceInput{
		OrganizationID: input.OrganizationID,
		AnalysisRunID:  input.AnalysisRunID,
		ProductID:      input.ProductID,
		ClaimID:        "recall-" + rec.SourceRecordID,
		Source:         rec.Source,
		SourceType:     sourceType,
		Claim:          "Product matched an official recall record",
		Snippet:        rec.Hazard,
		Reference:      rec.Reference,
		Reliability:    match.Confidence,
		Freshness:      freshness,
		RetrievedAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"sourceRecordId":  rec.SourceRecordID,
			"matchMethod":     match.Method,
			"matchConfidence": match.Confidence,
			"hazard":          rec.Hazard,
			"publishedAt":     rec.PublishedAt.UTC().Format(time.RFC3339),
			"safetyType":      string(domain.SafetyRecall),
		},
	})
}

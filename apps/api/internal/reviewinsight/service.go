package reviewinsight

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidInput = errors.New("invalid review analyzer input")

type Service struct {
	Insights *repository.ReviewInsightRepository
	Runs     *repository.AnalysisRunRepository
}

type AnalyzeInput struct {
	OrganizationID primitive.ObjectID
	AnalysisRunID  primitive.ObjectID
	ProductID      primitive.ObjectID
	ReviewIDs      []string
	Reviews        []domain.MarketplaceReview
	Experiences    []domain.UserExperience
}

type AnalyzeResult struct {
	PositiveSignals    []domain.ReviewSignal
	NegativeSignals    []domain.ReviewSignal
	SafetySignals      []domain.ReviewSignal
	QualityIssues      []domain.ReviewSignal
	DurabilityIssues   []domain.ReviewSignal
	UsabilityIssues    []domain.ReviewSignal
	AssemblyIssues     []domain.ReviewSignal
	AgeMismatchSignals []domain.ReviewSignal
	RepeatedComplaints []domain.ReviewSignal
	Issues             []string
	ReviewCount        int
}

func (s *Service) Analyze(ctx context.Context, input AnalyzeInput) (*AnalyzeResult, error) {
	if input.OrganizationID.IsZero() || input.AnalysisRunID.IsZero() || input.ProductID.IsZero() {
		return nil, ErrInvalidInput
	}
	if _, err := s.Runs.FindByID(ctx, input.OrganizationID, input.AnalysisRunID); err != nil {
		return nil, err
	}

	wanted := map[string]bool{}
	for _, id := range input.ReviewIDs {
		if n := strings.TrimSpace(id); n != "" {
			wanted[n] = true
		}
	}
	docs := make([]classifiedDoc, 0)
	for _, rv := range input.Reviews {
		if len(wanted) > 0 && !wanted[rv.ID.Hex()] {
			continue
		}
		if rv.OrganizationID != input.OrganizationID || rv.ProductID != input.ProductID {
			continue
		}
		docs = append(docs, classifyText(rv.ReviewText, rv.Rating))
		if len(docs) >= MaxReviews {
			break
		}
	}
	if len(docs) < MaxReviews {
		for _, ux := range input.Experiences {
			if ux.OrganizationID != input.OrganizationID || ux.ProductID != input.ProductID {
				continue
			}
			docs = append(docs, classifyText(ux.Narrative, nil))
			if len(docs) >= MaxReviews {
				break
			}
		}
	}

	result := aggregate(docs)
	result.ReviewCount = len(docs)
	if len(docs) == 0 {
		result.Issues = append(result.Issues, "NO_REVIEWS")
	}
	rec := &domain.ReviewInsightRecord{
		OrganizationID:     input.OrganizationID,
		AnalysisRunID:      input.AnalysisRunID,
		ProductID:          input.ProductID,
		ReviewCount:        result.ReviewCount,
		PositiveSignals:    result.PositiveSignals,
		NegativeSignals:    result.NegativeSignals,
		SafetySignals:      result.SafetySignals,
		QualityIssues:      result.QualityIssues,
		DurabilityIssues:   result.DurabilityIssues,
		UsabilityIssues:    result.UsabilityIssues,
		AssemblyIssues:     result.AssemblyIssues,
		AgeMismatchSignals: result.AgeMismatchSignals,
		RepeatedComplaints: result.RepeatedComplaints,
	}
	if err := s.Insights.Create(ctx, rec); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) LatestForRun(ctx context.Context, organizationID, analysisRunID primitive.ObjectID) (*domain.ReviewInsightRecord, error) {
	return s.Insights.LatestByAnalysisRun(ctx, organizationID, analysisRunID)
}

func aggregate(docs []classifiedDoc) *AnalyzeResult {
	out := &AnalyzeResult{
		PositiveSignals:    []domain.ReviewSignal{},
		NegativeSignals:    []domain.ReviewSignal{},
		SafetySignals:      []domain.ReviewSignal{},
		QualityIssues:      []domain.ReviewSignal{},
		DurabilityIssues:   []domain.ReviewSignal{},
		UsabilityIssues:    []domain.ReviewSignal{},
		AssemblyIssues:     []domain.ReviewSignal{},
		AgeMismatchSignals: []domain.ReviewSignal{},
		RepeatedComplaints: []domain.ReviewSignal{},
		Issues:             []string{},
	}
	pos, neg := 0, 0
	kindTopics := map[domain.ReviewSignalKind]map[string]int{}
	tokenCount := map[string]int{}
	for _, doc := range docs {
		if doc.positive {
			pos++
		}
		if doc.negative {
			neg++
		}
		for kind, topic := range doc.kinds {
			if kindTopics[kind] == nil {
				kindTopics[kind] = map[string]int{}
			}
			kindTopics[kind][topic]++
		}
		if doc.negative {
			for _, tok := range normalizeTokens(doc.text) {
				tokenCount[tok]++
			}
		}
	}
	if pos > 0 {
		out.PositiveSignals = append(out.PositiveSignals, domain.ReviewSignal{Topic: "positive", Count: pos, Summary: "Positive review or rating signals.", Kind: domain.ReviewSignalPositive})
	}
	if neg > 0 {
		out.NegativeSignals = append(out.NegativeSignals, domain.ReviewSignal{Topic: "negative", Count: neg, Summary: "Negative review or rating signals.", Kind: domain.ReviewSignalNegative})
	}
	out.SafetySignals = signalsFrom(kindTopics[domain.ReviewSignalSafety], domain.ReviewSignalSafety)
	out.QualityIssues = signalsFrom(kindTopics[domain.ReviewSignalQuality], domain.ReviewSignalQuality)
	out.DurabilityIssues = signalsFrom(kindTopics[domain.ReviewSignalDurability], domain.ReviewSignalDurability)
	out.UsabilityIssues = signalsFrom(kindTopics[domain.ReviewSignalUsability], domain.ReviewSignalUsability)
	out.AssemblyIssues = signalsFrom(kindTopics[domain.ReviewSignalAssembly], domain.ReviewSignalAssembly)
	out.AgeMismatchSignals = signalsFrom(kindTopics[domain.ReviewSignalAge], domain.ReviewSignalAge)
	for tok, n := range tokenCount {
		if n >= 2 {
			out.RepeatedComplaints = append(out.RepeatedComplaints, domain.ReviewSignal{Topic: tok, Count: n, Summary: "Repeated complaint theme.", Kind: domain.ReviewSignalRepeated})
		}
	}
	sort.Slice(out.RepeatedComplaints, func(i, j int) bool { return out.RepeatedComplaints[i].Count > out.RepeatedComplaints[j].Count })
	if len(out.RepeatedComplaints) > 8 {
		out.RepeatedComplaints = out.RepeatedComplaints[:8]
	}
	return out
}

func signalsFrom(topics map[string]int, kind domain.ReviewSignalKind) []domain.ReviewSignal {
	out := make([]domain.ReviewSignal, 0, len(topics))
	for topic, n := range topics {
		out = append(out, domain.ReviewSignal{Topic: topic, Count: n, Summary: string(kind) + " signal from reviews.", Kind: kind})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

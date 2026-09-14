package analysis

import (
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

type DecisionInput struct {
	Findings      []domain.SafetyFinding
	Matches       []domain.RecallMatchRecord
	Evidence      []domain.Evidence
	Validations   []domain.EvidenceClaimValidation
	Flags         []string
	ComplianceMax string
}

type DecisionResult struct {
	Decision       domain.ProductDecision
	OverallRisk    string
	Recommendation string
}

func Decide(in DecisionInput) DecisionResult {
	if confirmedCriticalRecall(in) {
		return DecisionResult{Decision: domain.DecisionBlock, OverallRisk: "CRITICAL", Recommendation: "Resmi ve doğrulanmış bir geri çağırma bu ürünle eşleşti. Güvenli kabul etmeyin."}
	}
	if highSeverityStrongEvidence(in) {
		return DecisionResult{Decision: domain.DecisionBlock, OverallRisk: "HIGH", Recommendation: "Yüksek şiddetli bir bulgu güçlü kanıtla destekleniyor."}
	}
	if hasFlag(in.Flags, string(domain.FlagContradictoryEvidence)) || contradictory(in.Validations) {
		return DecisionResult{Decision: domain.DecisionReviewRequired, OverallRisk: "MEDIUM", Recommendation: "Kaynaklar birbiriyle çelişiyor. Bir incelemenin kanıtı kontrol etmesi gerekir."}
	}
	if weakOrUncertainRisk(in) {
		return DecisionResult{Decision: domain.DecisionReviewRequired, OverallRisk: "MEDIUM", Recommendation: "Risk işaretleri var ancak eşleşme veya kanıt, otomatik engelleme için yeterince güçlü değil."}
	}
	if mediumSupportedWarning(in) {
		return DecisionResult{Decision: domain.DecisionAllowWithWarning, OverallRisk: "MEDIUM", Recommendation: "Desteklenen orta düzey sorunlar bulundu. Listelenen uyarılara dikkat edin."}
	}
	if strings.EqualFold(in.ComplianceMax, "BLOCK") && len(in.Findings) > 0 {
		return DecisionResult{Decision: domain.DecisionReviewRequired, OverallRisk: "MEDIUM", Recommendation: "Kuruluş politikası, herhangi bir güvenlik bulgusu olduğunda ek inceleme istiyor."}
	}
	return DecisionResult{Decision: domain.DecisionAllow, OverallRisk: "LOW", Recommendation: "Desteklenen önemli bir güvenlik riski doğrulanmadı."}
}

func confirmedCriticalRecall(in DecisionInput) bool {
	for _, f := range in.Findings {
		if f.Type == domain.SafetyRecall && f.Severity == domain.SafetySeverityCritical && len(f.EvidenceIDs) > 0 {
			return true
		}
	}
	for _, m := range in.Matches {
		if m.Matched && !m.RequiresReview && m.Confidence >= recall.ConfirmedMatchMinConfidence {
			return true
		}
	}
	return false
}

func highSeverityStrongEvidence(in DecisionInput) bool {
	for _, f := range in.Findings {
		if (f.Severity == domain.SafetySeverityHigh || f.Severity == domain.SafetySeverityCritical) &&
			f.Type != domain.SafetyInsufficientEvidence &&
			len(f.EvidenceIDs) > 0 &&
			f.Confidence >= 0.8 {
			if f.Type == domain.SafetyRecall && weakRecallOnly(in.Matches) {
				continue
			}
			return true
		}
	}
	return false
}

func weakOrUncertainRisk(in DecisionInput) bool {
	if weakRecallOnly(in.Matches) {
		return true
	}
	if hasFlag(in.Flags, string(domain.FlagInventedRecall), string(domain.FlagInsufficientEvidence), string(domain.FlagUnsupportedClaim)) {
		return true
	}
	for _, f := range in.Findings {
		if (f.Severity == domain.SafetySeverityHigh || f.Severity == domain.SafetySeverityCritical) &&
			(len(f.EvidenceIDs) == 0 || f.Type == domain.SafetyInsufficientEvidence || f.Confidence < 0.8) {
			return true
		}
	}
	return false
}

func mediumSupportedWarning(in DecisionInput) bool {
	for _, f := range in.Findings {
		if f.Severity == domain.SafetySeverityMedium && len(f.EvidenceIDs) > 0 {
			return true
		}
	}
	for _, ev := range in.Evidence {
		if ev.Reliability >= 0.7 && ev.SourceType != "OFFICIAL_RECALL" {
			return strings.Contains(strings.ToLower(ev.Claim), "warning") || strings.Contains(strings.ToLower(ev.Claim), "uyarı")
		}
	}
	for _, v := range in.Validations {
		if v.SupportStatus == domain.EvidencePartiallySupported || v.SupportStatus == domain.EvidenceSupported {
			if strings.Contains(strings.ToLower(v.ClaimText), "warning") || strings.Contains(strings.ToLower(v.ClaimText), "kalite") {
				return true
			}
		}
	}
	return false
}

func weakRecallOnly(matches []domain.RecallMatchRecord) bool {
	sawWeak := false
	for _, m := range matches {
		if m.Matched && !m.RequiresReview && m.Confidence >= recall.ConfirmedMatchMinConfidence {
			return false
		}
		if m.RequiresReview || m.Method == recall.MethodFuzzyName {
			sawWeak = true
		}
	}
	return sawWeak
}

func contradictory(vals []domain.EvidenceClaimValidation) bool {
	for _, v := range vals {
		if v.SupportStatus == domain.EvidenceContradicted {
			return true
		}
	}
	return false
}

package marketplace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportStats struct {
	ProductsAccepted   int
	ReviewsAccepted    int
	ReviewsSkipped     int
	RecordsQuarantined int
}

func resolveImportGovernance(result *importpkg.ParseResult, reviewRow importpkg.ReviewRow) (
	domain.ProvenanceStatus,
	domain.LicenseStatus,
	domain.UsageRightsStatus,
	domain.DatasetEligibility,
	domain.PIIStatus,
	domain.ModerationStatus,
	domain.QualityStatus,
) {
	provenance := domain.ProvenancePartial
	license := domain.LicenseUnknown
	usage := domain.UsageRightsUnknown
	eligibility := domain.EligibilityQuarantined
	pii := domain.PIIClean
	moderation := domain.ModerationPending
	quality := domain.QualityPending

	if reviewRow.PIIStatus != "" {
		pii = reviewRow.PIIStatus
	}
	if reviewRow.Governance != nil {
		g := reviewRow.Governance
		provenance = g.ProvenanceStatus
		license = g.LicenseStatus
		usage = g.UsageRightsStatus
		eligibility = g.DatasetEligibility
	}

	if result != nil && result.Source == importpkg.AmazonBabyDatasetSource {
		g := importpkg.AmazonBabyDefaultGovernance()
		provenance = g.ProvenanceStatus
		license = g.LicenseStatus
		usage = g.UsageRightsStatus
		eligibility = g.DatasetEligibility
	}
	return provenance, license, usage, eligibility, pii, moderation, quality
}

func (s *Service) importParseResult(ctx context.Context, organizationID, actorID primitive.ObjectID, source domain.MarketplaceSource, result *importpkg.ParseResult) (*ImportStats, error) {
	stats := &ImportStats{}
	if result == nil || len(result.Products) == 0 {
		return stats, nil
	}

	importSource := source
	if result.Source == importpkg.AmazonBabyDatasetSource {
		importSource = domain.MarketplaceOther
	}

	for _, row := range result.Products {
		sourceProductID := row.SourceProductID
		if sourceProductID == "" {
			sourceProductID = importpkg.NormalizeProductKey(row.Name)
		}

		sourceURL := ""
		if result.Source == importpkg.AmazonBabyDatasetSource {
			sourceURL = importpkg.AmazonBabySourceURL
		}

		prod, _, err := s.Products.Create(ctx, product.CreateInput{
			OrganizationID:  organizationID,
			ActorID:         actorID,
			Name:            row.Name,
			Brand:           row.Brand,
			Category:        row.Category,
			Description:     row.Description,
			Source:          importSource,
			SourceProductID: sourceProductID,
			SourceURL:       sourceURL,
			SKU:             row.SKU,
		})
		if err != nil {
			continue
		}
		stats.ProductsAccepted++

		for _, reviewRow := range row.Reviews {
			text, lang, fp := importpkg.NormalizeReviewRow(reviewRow, importSource, sourceProductID)
			if text == "" {
				stats.ReviewsSkipped++
				continue
			}
			if _, err := s.Reviews.FindByFingerprint(ctx, organizationID, fp); err == nil {
				stats.ReviewsSkipped++
				continue
			} else if !errors.Is(err, repository.ErrNotFound) {
				return stats, err
			}

			provenance, license, usage, eligibility, pii, moderation, quality := resolveImportGovernance(result, reviewRow)

			rev := &domain.MarketplaceReview{
				OrganizationID:     organizationID,
				ProductID:          prod.ID,
				SourceType:         domain.SourceTypeMarketplaceReview,
				Source:             importSource,
				SourceProductID:    sourceProductID,
				SourceReviewID:     reviewRow.SourceReviewID,
				Rating:             reviewRow.Rating,
				ReviewText:         text,
				FetchedAt:          time.Now().UTC(),
				Language:           lang,
				PIIStatus:          pii,
				ModerationStatus:   moderation,
				SpamStatus:         domain.SpamClean,
				DuplicateStatus:    domain.DuplicateUnique,
				QualityStatus:      quality,
				ProvenanceStatus:   provenance,
				LicenseStatus:      license,
				UsageRightsStatus:  usage,
				DatasetEligibility: eligibility,
				Fingerprint:        fp,
			}
			if result.Source == importpkg.AmazonBabyDatasetSource {
				rev.SourceURL = importpkg.AmazonBabySourceURL
			}
			if err := s.Reviews.Create(ctx, rev); err != nil {
				if errors.Is(err, repository.ErrDuplicate) {
					stats.ReviewsSkipped++
					continue
				}
				return stats, err
			}
			stats.ReviewsAccepted++
			if eligibility == domain.EligibilityQuarantined {
				stats.RecordsQuarantined++
			}

			if s.DatasetRecords != nil {
				record := &domain.DatasetRecord{
					OrganizationID:     organizationID,
					RecordType:         domain.RecordMarketplaceReview,
					SourceRecordID:     rev.ID,
					ProductID:          prod.ID,
					DatasetEligibility: eligibility,
					ProvenanceStatus:   provenance,
					LicenseStatus:      license,
					UsageRightsStatus:  usage,
					QualityStatus:      quality,
					PIIStatus:          pii,
				}
				if err := s.DatasetRecords.Create(ctx, record); err != nil {
					return stats, err
				}
			}
		}
	}
	return stats, nil
}

func parseObjectFromImportRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty import ref")
	}
	if strings.HasPrefix(ref, "s3://") {
		parts := strings.SplitN(strings.TrimPrefix(ref, "s3://"), "/", 2)
		if len(parts) != 2 || parts[1] == "" {
			return "", fmt.Errorf("invalid s3 ref")
		}
		return parts[1], nil
	}
	return ref, nil
}

func parseImportPayload(content []byte, filename string) (*importpkg.ParseResult, error) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".jsonl"):
		return importpkg.ParseMiyunaJSONL(bytes.NewReader(content))
	case strings.HasSuffix(lower, ".json"):
		if importpkg.LooksLikeMiyunaJSONL(content) {
			return importpkg.ParseMiyunaJSONL(bytes.NewReader(content))
		}
		return importpkg.ParseJSON(bytes.NewReader(content))
	default:
		return importpkg.ParseCSV(bytes.NewReader(content))
	}
}

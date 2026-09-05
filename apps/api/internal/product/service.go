package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/rbac"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrForbidden = errors.New("forbidden")

type Service struct {
	Products *repository.ProductRepository
	Mappings *repository.ProductSourceMappingRepository
	Security *repository.SecurityEventRepository
	Tenant   *tenant.Guard
}

type CreateInput struct {
	OrganizationID  primitive.ObjectID
	ActorID         primitive.ObjectID
	Name            string
	Brand           string
	Category        string
	Description     string
	Source          domain.MarketplaceSource
	SourceProductID string
	SourceURL       string
	GTIN            string
	EAN             string
	UPC             string
	SKU             string
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*domain.Product, bool, error) {
	role, err := s.Tenant.RequireMembership(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return nil, false, err
	}
	if !rbac.CanManageProducts(role) {
		return nil, false, ErrForbidden
	}

	sourceProductID := normalize.NormalizeIdentifier(input.SourceProductID)
	if sourceProductID != "" && input.Source != "" {
		existing, err := s.Mappings.FindBySourceProduct(ctx, input.OrganizationID, input.Source, sourceProductID)
		if err == nil {
			product, err := s.Products.FindByID(ctx, input.OrganizationID, existing.ProductID)
			return product, true, err
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, false, err
		}
	}

	normalized := normalize.NormalizeProductInput(normalize.ProductInput{
		Name:        input.Name,
		Brand:       input.Brand,
		Category:    input.Category,
		Description: input.Description,
		SKU:         input.SKU,
	}, string(input.Source), sourceProductID)

	product := &normalized
	product.OrganizationID = input.OrganizationID
	product.CreatedBy = input.ActorID
	if err := s.Products.Create(ctx, product); err != nil {
		return nil, false, err
	}

	mapping := &domain.ProductSourceMapping{
		OrganizationID:  input.OrganizationID,
		ProductID:       product.ID,
		Source:          input.Source,
		SourceProductID: sourceProductID,
		SourceURL:       strings.TrimSpace(input.SourceURL),
		GTIN:            normalize.NormalizeIdentifier(input.GTIN),
		EAN:             normalize.NormalizeIdentifier(input.EAN),
		UPC:             normalize.NormalizeIdentifier(input.UPC),
		SKU:             normalize.NormalizeIdentifier(input.SKU),
		Brand:           fmt.Sprint(product.Brand.Value),
		MatchStatus:     domain.MatchVerified,
		MatchMethod:     domain.MatchExplicit,
		Confidence:      1.0,
	}
	if err := s.Mappings.Create(ctx, mapping); err != nil && !errors.Is(err, repository.ErrDuplicate) {
		return nil, false, err
	}

	uid := input.ActorID
	oid := input.OrganizationID
	_ = s.Security.Record(ctx, domain.SecurityEvent{
		OrganizationID: &oid,
		UserID:         &uid,
		EventType:      domain.EventProductCreated,
		Severity:       domain.SeverityInfo,
		Details:        map[string]string{"productId": product.ID.Hex()},
	})
	return product, false, nil
}

func (s *Service) Get(ctx context.Context, actorID, organizationID, productID primitive.ObjectID) (*domain.Product, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadProducts(role) {
		return nil, ErrForbidden
	}
	return s.Products.FindByID(ctx, organizationID, productID)
}

func (s *Service) List(ctx context.Context, actorID, organizationID primitive.ObjectID, limit int) ([]domain.Product, error) {
	role, err := s.Tenant.RequireMembership(ctx, actorID, organizationID)
	if err != nil {
		return nil, err
	}
	if !rbac.CanReadProducts(role) {
		return nil, ErrForbidden
	}
	return s.Products.ListByOrg(ctx, organizationID, limit)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

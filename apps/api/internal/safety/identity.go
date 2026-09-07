package safety

import (
	"fmt"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/normalize"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/safety/recall"
)

func IdentityFromProduct(product *domain.Product, mappings []domain.ProductSourceMapping) recall.Identity {
	id := recall.Identity{}
	if product != nil {
		id.Name = stringifyField(product.Name.Value)
		id.Brand = stringifyField(product.Brand.Value)
		id.Model = normalize.NormalizeIdentifier(stringifyField(product.SKU.Value))
	}
	for _, m := range mappings {
		if m.SourceProductID != "" {
			id.SourceIDs = append(id.SourceIDs, normalize.NormalizeIdentifier(m.SourceProductID))
		}
		if m.GTIN != "" {
			id.GTIN = normalize.NormalizeIdentifier(m.GTIN)
		}
		if m.UPC != "" {
			id.UPC = normalize.NormalizeIdentifier(m.UPC)
		}
		if m.EAN != "" {
			id.EAN = normalize.NormalizeIdentifier(m.EAN)
		}
		if m.Model != "" {
			id.Model = normalize.NormalizeIdentifier(m.Model)
		}
		if m.Brand != "" && id.Brand == "" {
			id.Brand = m.Brand
		}
	}
	id.Manufacturer = id.Brand
	return id
}

func stringifyField(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

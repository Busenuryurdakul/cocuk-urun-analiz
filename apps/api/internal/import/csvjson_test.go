package importpkg_test

import (
	"strings"
	"testing"

	importpkg "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/import"
)

func TestParseJSONProducts(t *testing.T) {
	payload := `{"products":[{"name":"Toy","brand":"Acme","sourceProductId":"1","reviews":[{"reviewText":"good"}]}]}`
	result, err := importpkg.ParseJSON(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(result.Products) != 1 || result.Products[0].Name != "Toy" {
		t.Fatalf("unexpected parse result: %+v", result)
	}
}

func TestParseCSVProducts(t *testing.T) {
	payload := "name,brand,source_product_id\nToy,Acme,1\n"
	result, err := importpkg.ParseCSV(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(result.Products) != 1 || result.Products[0].Brand != "Acme" {
		t.Fatalf("unexpected parse result: %+v", result)
	}
}

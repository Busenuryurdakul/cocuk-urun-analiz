package graph

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

func TestCreateProductInputAcceptsManualCatalogFields(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "schema.graphqls"))
	if err != nil {
		t.Fatal(err)
	}
	schema := gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphqls", Input: string(src)})
	doc, errs := gqlparser.LoadQuery(schema, `mutation($input: CreateProductInput!) {
		createProduct(input: $input) { product { id } }
	}`)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	_, err = validator.VariableValues(schema, doc.Operations[0], map[string]any{
		"input": map[string]any{
			"organizationId":  "org",
			"name":            "Ahşap çıngırak",
			"source":          "MIYUNA",
			"sourceProductId": "ahsap-cingirak",
			"targetAge":       "0-3",
			"materials":       "ahşap",
			"safetyWarnings":  "küçük parça",
			"currentPrice":    "199",
			"originalPrice":   "249",
			"currency":        "TRY",
			"seller":          "Miyuna",
			"rating":          "4.5",
			"reviewCount":     "12",
			"stockStatus":     "in_stock",
		},
	})
	if err != nil {
		t.Fatalf("variable coercion: %v", err)
	}
}


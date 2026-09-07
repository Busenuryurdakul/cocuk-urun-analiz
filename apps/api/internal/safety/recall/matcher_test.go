package recall

import "testing"

func TestMatchExactIdentifiers(t *testing.T) {
	rec := Record{SourceRecordID: "R-1", GTIN: "01234567890123", UPC: "012345678905", EAN: "4006381333931", Brand: "Acme", Model: "X1", ProductName: "Baby Rattle"}
	if !Confirmed(Match(Identity{SourceIDs: []string{"R-1"}}, rec)) {
		t.Fatal("exact source id")
	}
	if got := Match(Identity{GTIN: "01234567890123"}, rec); !Confirmed(got) || got.Method != MethodGTIN {
		t.Fatalf("gtin: %+v", got)
	}
	if got := Match(Identity{UPC: "012345678905"}, rec); !Confirmed(got) || got.Method != MethodUPC {
		t.Fatalf("upc: %+v", got)
	}
	if got := Match(Identity{EAN: "4006381333931"}, rec); !Confirmed(got) || got.Method != MethodEAN {
		t.Fatalf("ean: %+v", got)
	}
}

func TestMatchBrandAndManufacturerModel(t *testing.T) {
	rec := Record{Manufacturer: "Acme", Brand: "Acme", Model: "X1", ProductName: "Baby Rattle"}
	if got := Match(Identity{Manufacturer: "Acme", Model: "X1"}, rec); !Confirmed(got) || got.Method != MethodManufacturerModel {
		t.Fatalf("manufacturer+model: %+v", got)
	}
	if got := Match(Identity{Brand: "Acme", Model: "X1"}, rec); !Confirmed(got) || got.Method != MethodBrandModel {
		t.Fatalf("brand+model: %+v", got)
	}
	if got := Match(Identity{Brand: "Acme", Name: "Baby Rattle"}, rec); Confirmed(got) || !got.RequiresReview {
		t.Fatalf("brand+name must require review: %+v", got)
	}
}

func TestMatchFuzzyAndFalsePositives(t *testing.T) {
	rec := Record{Brand: "Acme", Model: "X1", ProductName: "Wooden Baby Rattle Toy"}
	if got := Match(Identity{Name: "wooden rattle toy"}, rec); Confirmed(got) || !got.RequiresReview {
		t.Fatalf("fuzzy must not be confirmed: %+v", got)
	}
	if got := Match(Identity{Brand: "OtherCo", Name: "Wooden Baby Rattle Toy"}, rec); got.Matched || Confirmed(got) {
		t.Fatalf("same name different brand: %+v", got)
	}
	if got := Match(Identity{Brand: "Acme", Model: "Y9"}, rec); got.Matched || Confirmed(got) {
		t.Fatalf("same brand different model: %+v", got)
	}
	if got := Match(Identity{Name: "garden hose"}, rec); Confirmed(got) {
		t.Fatalf("unrelated name: %+v", got)
	}
	if got := Match(Identity{}, rec); got.Matched {
		t.Fatalf("empty identity: %+v", got)
	}
}

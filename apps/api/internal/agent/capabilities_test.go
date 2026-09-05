package agent

import (
	"testing"
)

func TestRegistryVersionDeterministic(t *testing.T) {
	reg := NewRegistry()
	v1 := RegistryVersion(reg)
	v2 := RegistryVersion(reg)
	if v1 != v2 {
		t.Fatalf("registry version not deterministic: %s vs %s", v1, v2)
	}
	if v1 == "" || v1 == "1.0.0" {
		t.Fatalf("unexpected registry version: %s", v1)
	}
}

func TestCapabilitySnapshotExcludesUnavailableTools(t *testing.T) {
	svc := &Service{Registry: NewRegistry()}
	snap := svc.BuildCapabilitySnapshot()
	for _, tool := range snap.AvailableTools {
		if tool.Availability != string(ToolAvailable) {
			t.Fatalf("non-available tool in snapshot: %s", tool.Name)
		}
		if tool.Name == "import_planner" || tool.Name == "product_normalizer" || tool.Name == "ecommerce_fetcher" {
			t.Fatalf("phase4/import tool must not appear available: %s", tool.Name)
		}
	}
}

func TestPythonPlannerVersionLabel(t *testing.T) {
	if PythonPlannerVersion() == "" {
		t.Fatal("planner version required")
	}
}

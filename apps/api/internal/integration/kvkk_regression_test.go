package integration

import (
	"strings"
	"testing"
)

func TestKVKKExportAndDeletionGraphQLSelfScoped(t *testing.T) {
	h := NewPhase5Harness(t)
	token := h.AccessToken(h.AnalystID, h.OrgID)

	exportData, exportErrs, _ := h.GraphQL(token, `mutation {
		exportMyData { schemaVersion exportedAt payload }
	}`, map[string]any{})
	if len(exportErrs) > 0 {
		t.Fatalf("exportMyData: %v", exportErrs)
	}
	exp := exportData["exportMyData"].(map[string]any)
	if exp["schemaVersion"] != "1.0.0" {
		t.Fatalf("schemaVersion=%v", exp["schemaVersion"])
	}
	payload, _ := exp["payload"].(string)
	if strings.Contains(payload, "passwordHash") || strings.Contains(payload, "mfaSecret") {
		t.Fatal("export payload must not include credential fields")
	}

	reqData, reqErrs, _ := h.GraphQL(token, `mutation {
		requestAccountDeletion { status expiresAt }
	}`, map[string]any{})
	if len(reqErrs) > 0 {
		t.Fatalf("requestAccountDeletion: %v", reqErrs)
	}
	if reqData["requestAccountDeletion"].(map[string]any)["status"] != "PENDING_CONFIRMATION" {
		t.Fatal("expected PENDING_CONFIRMATION")
	}

	_, unauthErrs, _ := h.GraphQL("", `mutation { exportMyData { schemaVersion } }`, map[string]any{})
	if !errHasCode(unauthErrs, "UNAUTHORIZED") && len(unauthErrs) == 0 {
		t.Fatalf("expected unauthorized export, got %v", unauthErrs)
	}
}

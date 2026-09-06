package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestHarnessLLMCompleteEndpoint(t *testing.T) {
	h := NewPhase5Harness(t)
	body, _ := json.Marshal(map[string]any{
		"organizationId": h.OrgID.Hex(),
		"analysisRunId":  h.OrgID.Hex(),
		"correlationId":  "harness-llm-test",
		"idempotencyKey": "harness-llm-test",
		"taskType":       "analysis",
		"personaKey":     domain.PersonaCarefulAnalyst,
		"userPrompt":     "Summarize product evidence.",
	})
	req, err := http.NewRequestWithContext(h.Ctx, http.MethodPost, h.BaseURL+"/internal/llm/v1/complete", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Miyuna-Internal-Token", internalToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200 got %d body=%s", resp.StatusCode, buf.String())
	}
}

package llm

import "testing"

func TestProviderAllowsEmptyAPIKeyForLocalhost(t *testing.T) {
	cases := []string{
		"http://127.0.0.1:11434/v1",
		"http://localhost:1234/v1",
		"http://127.0.0.1:8080/v1",
	}
	for _, baseURL := range cases {
		if !ProviderAllowsEmptyAPIKey(baseURL) {
			t.Fatalf("expected empty api key allowed for %q", baseURL)
		}
		if !ProviderCredentialsConfigured(baseURL, "") {
			t.Fatalf("expected configured with empty api key for %q", baseURL)
		}
	}
}

func TestProviderRequiresAPIKeyForPublicHost(t *testing.T) {
	baseURL := "https://router.huggingface.co/v1"
	if ProviderAllowsEmptyAPIKey(baseURL) {
		t.Fatal("public host must not allow empty api key")
	}
	if ProviderCredentialsConfigured(baseURL, "") {
		t.Fatal("public host requires api key")
	}
	if !ProviderCredentialsConfigured(baseURL, "hf_test") {
		t.Fatal("public host with api key should be configured")
	}
}

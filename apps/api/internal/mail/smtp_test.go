package mail

import (
	"strings"
	"testing"
)

func TestParseFrom(t *testing.T) {
	tests := []struct {
		raw      string
		envelope string
		header   string
	}{
		{"onboarding@resend.dev", "onboarding@resend.dev", "onboarding@resend.dev"},
		{"Miyuna <onboarding@resend.dev>", "onboarding@resend.dev", "Miyuna <onboarding@resend.dev>"},
	}
	for _, tc := range tests {
		env, hdr := parseFrom(tc.raw)
		if env != tc.envelope || hdr != tc.header {
			t.Fatalf("parseFrom(%q) = (%q, %q), want (%q, %q)", tc.raw, env, hdr, tc.envelope, tc.header)
		}
	}
}

func TestBuildMIMEIncludesFrom(t *testing.T) {
	body := buildMIME("Miyuna <onboarding@resend.dev>", Message{
		To:      "user@example.com",
		Subject: "Test",
		Body:    "hello",
	})
	if !strings.Contains(body, "From: Miyuna <onboarding@resend.dev>") {
		t.Fatalf("missing From header in MIME: %q", body)
	}
}

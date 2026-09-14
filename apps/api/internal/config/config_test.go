package config

import "testing"

func TestLocalRelaxedAuthDefaults(t *testing.T) {
	t.Setenv("AUTH_LOCAL_RELAXED", "")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("JWT_SECRET", "change-me-jwt-dev-secret-32chars")
	if !localRelaxedAuth() {
		t.Fatal("expected local relaxed with stock defaults")
	}

	t.Setenv("COOKIE_SECURE", "true")
	if localRelaxedAuth() {
		t.Fatal("COOKIE_SECURE=true must disable relaxed login")
	}

	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("JWT_SECRET", "production-secret-not-the-default")
	if localRelaxedAuth() {
		t.Fatal("custom JWT secret must disable relaxed login")
	}

	t.Setenv("AUTH_LOCAL_RELAXED", "false")
	t.Setenv("JWT_SECRET", "change-me-jwt-dev-secret-32chars")
	if localRelaxedAuth() {
		t.Fatal("explicit AUTH_LOCAL_RELAXED=false must win")
	}

	t.Setenv("AUTH_LOCAL_RELAXED", "true")
	t.Setenv("COOKIE_SECURE", "true")
	if !localRelaxedAuth() {
		t.Fatal("explicit AUTH_LOCAL_RELAXED=true must win")
	}
}

func TestCORSAllowedOriginsIncludesAltSite(t *testing.T) {
	t.Setenv("WEB_BASE_URL", "https://miyuna-web.vercel.app")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://preview.example")
	cfg := Load()
	got := cfg.CORSAllowedOrigins()
	want := []string{
		"https://miyuna-web.vercel.app",
		"https://preview.example",
		"https://miyuna-web-alt.vercel.app",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}
	if len(got) != len(want) {
		t.Fatalf("origins=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("origins[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

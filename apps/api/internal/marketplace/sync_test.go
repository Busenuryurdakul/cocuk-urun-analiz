package marketplace

import (
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
)

func TestIsSyncCooldownActive(t *testing.T) {
	now := time.Now().UTC()
	if !isSyncCooldownActive(&now, time.Hour, false) {
		t.Fatalf("expected cooldown active")
	}
	if isSyncCooldownActive(&now, time.Hour, true) {
		t.Fatalf("force should bypass cooldown")
	}
	old := now.Add(-2 * time.Hour)
	if isSyncCooldownActive(&old, time.Hour, false) {
		t.Fatalf("expected cooldown expired")
	}
}

func TestDeferredWarningMessage(t *testing.T) {
	if deferredWarning(domain.DeferredFetchReason) != warningDeferredNoAPI {
		t.Fatalf("unexpected deferred warning")
	}
}

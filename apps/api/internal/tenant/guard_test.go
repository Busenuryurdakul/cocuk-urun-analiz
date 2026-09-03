package tenant_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestTenantEscapeCrossOrgAccess(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017/miyuna_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.DB
	members := repository.NewMemberRepository(db)
	events := repository.NewSecurityEventRepository(db)
	guard := &tenant.Guard{Members: members, Events: events}

	userA := primitive.NewObjectID()
	userB := primitive.NewObjectID()
	orgA := primitive.NewObjectID()

	if err := members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: orgA,
		UserID:         userA,
		Role:           domain.RoleOwner,
	}); err != nil {
		t.Fatal(err)
	}

	before, _ := events.CountByType(ctx, domain.EventCrossTenantAccess)
	_, err = guard.RequireMembership(ctx, userB, orgA)
	if err == nil {
		t.Fatal("expected cross-tenant access to be rejected")
	}
	after, _ := events.CountByType(ctx, domain.EventCrossTenantAccess)
	if after != before+1 {
		t.Fatalf("expected security event, before=%d after=%d", before, after)
	}
}

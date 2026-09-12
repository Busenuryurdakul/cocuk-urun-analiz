package llm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestActiveConfigurationMissingReturnsNil(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	uri := fmt.Sprintf("mongodb://localhost:27017/miyuna_llm_active_cfg_%d", time.Now().UnixNano())
	client, err := mongoclient.Connect(ctx, uri)
	if err != nil {
		t.Skipf("mongodb unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	db := client.DB
	members := repository.NewMemberRepository(db)
	security := repository.NewSecurityEventRepository(db)
	orgID := primitive.NewObjectID()
	owner := primitive.NewObjectID()
	if err := members.Create(ctx, &domain.OrganizationMember{
		OrganizationID: orgID, UserID: owner, Role: domain.RoleOwner,
	}); err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		OrgSettings: repository.NewLLMOrgSettingsRepository(db),
		Snapshots:   repository.NewConfigSnapshotRepository(db),
		Tenant:      &tenant.Guard{Members: members, Events: security},
	}

	snap, err := svc.ActiveConfiguration(ctx, owner, orgID)
	if err != nil {
		t.Fatalf("expected no error for missing config, got %v", err)
	}
	if snap != nil {
		t.Fatalf("expected nil snapshot, got %#v", snap)
	}
}

package llm

import (
	"context"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
)

func testRouterWithMiniRedis(t *testing.T) (*Router, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client, err := redis.Connect(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		mr.Close()
		t.Fatalf("redis connect: %v", err)
	}
	return &Router{Redis: client}, func() {
		_ = client.Close()
		mr.Close()
	}
}

func TestRouterRoundRobinAmongEqualLoad(t *testing.T) {
	router, cleanup := testRouterWithMiniRedis(t)
	defer cleanup()

	ctx := context.Background()
	policy := &domain.LLMRoutingPolicy{
		Version:          domain.LLMPlatformDefaultRoutingVersion,
		DefaultModelKey:  ModelKeyCareful,
		FallbackModelKey: ModelKeyResult,
		Rules: []domain.LLMRoutingRule{
			{TaskType: "analysis", PreferredModels: []string{ModelKeyCareful, ModelKeyResult}},
		},
	}
	models := []domain.LLMModel{
		{ModelKey: ModelKeyCareful, Status: domain.LLMModelActive, HealthStatus: domain.LLMHealthHealthy},
		{ModelKey: ModelKeyResult, Status: domain.LLMModelActive, HealthStatus: domain.LLMHealthHealthy},
	}

	first, err := router.Decide(ctx, RouteRequest{TaskType: "analysis", Policy: policy, Models: models})
	if err != nil {
		t.Fatalf("first decide: %v", err)
	}
	second, err := router.Decide(ctx, RouteRequest{TaskType: "analysis", Policy: policy, Models: models})
	if err != nil {
		t.Fatalf("second decide: %v", err)
	}
	if first.PrimaryModelKey == second.PrimaryModelKey {
		t.Fatalf("expected round-robin alternate, got %s twice", first.PrimaryModelKey)
	}
}

func TestRouterConcurrentLoadFairness(t *testing.T) {
	router, cleanup := testRouterWithMiniRedis(t)
	defer cleanup()

	ctx := context.Background()
	policy := &domain.LLMRoutingPolicy{
		Version:          domain.LLMPlatformDefaultRoutingVersion,
		DefaultModelKey:  ModelKeyCareful,
		FallbackModelKey: ModelKeyResult,
		Rules: []domain.LLMRoutingRule{
			{TaskType: "analysis", PreferredModels: []string{ModelKeyCareful, ModelKeyResult}},
		},
	}
	models := []domain.LLMModel{
		{ModelKey: ModelKeyCareful, Status: domain.LLMModelActive, HealthStatus: domain.LLMHealthHealthy},
		{ModelKey: ModelKeyResult, Status: domain.LLMModelActive, HealthStatus: domain.LLMHealthHealthy},
	}

	var wg sync.WaitGroup
	picks := make(chan string, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			decision, err := router.Decide(ctx, RouteRequest{TaskType: "analysis", Policy: policy, Models: models})
			if err != nil {
				t.Errorf("decide: %v", err)
				return
			}
			router.IncLoad(ctx, decision.PrimaryModelKey)
			defer router.DecLoad(ctx, decision.PrimaryModelKey)
			picks <- decision.PrimaryModelKey
		}()
	}
	wg.Wait()
	close(picks)

	counts := map[string]int{}
	for pick := range picks {
		counts[pick]++
	}
	if len(counts) != 2 {
		t.Fatalf("expected both models selected, got %#v", counts)
	}
	for key, n := range counts {
		if n < 1 {
			t.Fatalf("model %s never selected: %#v", key, counts)
		}
	}
}

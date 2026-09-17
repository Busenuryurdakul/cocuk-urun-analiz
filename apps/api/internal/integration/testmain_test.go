package integration

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pingMongo(ctx, "mongodb://localhost:27017/miyuna_integration_probe"); err == nil {
		if err := flushRedisDB(ctx, "redis://localhost:6379/15"); err == nil {
			integrationHarnessEnabled = true
		}
	}
	os.Exit(m.Run())
}

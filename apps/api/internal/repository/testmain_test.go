package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
)

var mongoTestsEnabled bool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := mongoclient.Connect(ctx, "mongodb://localhost:27017/miyuna_test_probe"); err == nil {
		mongoTestsEnabled = true
	}
	os.Exit(m.Run())
}

package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigAuditRepository struct {
	col *mongo.Collection
}

func NewConfigAuditRepository(db *mongo.Database) *ConfigAuditRepository {
	return &ConfigAuditRepository{col: db.Collection("config_audit_log")}
}

func (r *ConfigAuditRepository) Record(ctx context.Context, entry domain.ConfigAuditLog) error {
	entry.ChangedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, entry)
	return err
}

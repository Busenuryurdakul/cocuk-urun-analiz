package repository

import (
	"context"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

type ComplianceEventRepository struct {
	col *mongo.Collection
}

func NewComplianceEventRepository(db *mongo.Database) *ComplianceEventRepository {
	return &ComplianceEventRepository{col: db.Collection("compliance_events")}
}

func (r *ComplianceEventRepository) Record(ctx context.Context, event domain.ComplianceEvent) error {
	event.Timestamp = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, event)
	return err
}

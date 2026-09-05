package marketplace

import (
	"fmt"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseJobIDs(job queue.ImportJob) (primitive.ObjectID, primitive.ObjectID, error) {
	orgID, err := primitive.ObjectIDFromHex(job.OrganizationID)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, fmt.Errorf("org id: %w", err)
	}
	runID, err := primitive.ObjectIDFromHex(job.ImportRunID)
	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, fmt.Errorf("run id: %w", err)
	}
	return orgID, runID, nil
}

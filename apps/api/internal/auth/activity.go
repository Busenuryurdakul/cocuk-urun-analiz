package auth

import (
	"context"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/domain"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/httpx"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityLogger struct {
	Logs *repository.UserActivityLogRepository
}

func (a *ActivityLogger) Record(ctx context.Context, userID primitive.ObjectID, action string, deviceID *primitive.ObjectID, orgID *primitive.ObjectID, resourceID string) {
	if a == nil || a.Logs == nil {
		return
	}
	info := httpx.ClientInfoFrom(ctx)
	_ = a.Logs.Create(ctx, &domain.UserActivityLog{
		UserID:         userID,
		DeviceID:       deviceID,
		OrganizationID: orgID,
		Action:         action,
		ResourceID:     resourceID,
		IPAddress:      info.IPAddress,
		UserAgent:      info.UserAgent,
	})
}

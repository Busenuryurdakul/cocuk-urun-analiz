package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const runLeaseKeyPrefix = "agent:run:lease:"

type RunLease struct {
	OwnerID       string `json:"ownerId"`
	Attempt       int    `json:"attempt"`
	LastHeartbeat int64  `json:"lastHeartbeatAt"`
}

type LeaseStore struct {
	Redis *redis.Client
	TTL   time.Duration
}

func (l *LeaseStore) leaseKey(orgID, runID primitive.ObjectID) string {
	return runLeaseKeyPrefix + orgID.Hex() + ":" + runID.Hex()
}

func (l *LeaseStore) Claim(ctx context.Context, orgID, runID primitive.ObjectID, ownerID string, attempt int) (bool, error) {
	if l == nil || l.Redis == nil {
		return false, ErrGrantStoreUnavailable
	}
	ttl := l.TTL
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	payload, _ := json.Marshal(RunLease{
		OwnerID:       ownerID,
		Attempt:       attempt,
		LastHeartbeat: time.Now().UTC().Unix(),
	})
	return l.Redis.SetNX(ctx, l.leaseKey(orgID, runID), string(payload), ttl)
}

func (l *LeaseStore) Heartbeat(ctx context.Context, orgID, runID primitive.ObjectID, ownerID string) error {
	if l == nil || l.Redis == nil {
		return ErrGrantStoreUnavailable
	}
	key := l.leaseKey(orgID, runID)
	raw, err := l.Redis.Get(ctx, key)
	if err != nil || raw == "" {
		return fmt.Errorf("lease not held")
	}
	var lease RunLease
	if err := json.Unmarshal([]byte(raw), &lease); err != nil {
		return err
	}
	if lease.OwnerID != ownerID {
		return fmt.Errorf("lease owner mismatch")
	}
	lease.LastHeartbeat = time.Now().UTC().Unix()
	payload, _ := json.Marshal(lease)
	ttl := l.TTL
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return l.Redis.Set(ctx, key, string(payload), ttl)
}

func (l *LeaseStore) Release(ctx context.Context, orgID, runID primitive.ObjectID) error {
	if l == nil || l.Redis == nil {
		return nil
	}
	return l.Redis.Del(ctx, l.leaseKey(orgID, runID))
}

func (l *LeaseStore) Exists(ctx context.Context, orgID, runID primitive.ObjectID) (bool, error) {
	if l == nil || l.Redis == nil {
		return false, ErrGrantStoreUnavailable
	}
	return l.Redis.Exists(ctx, l.leaseKey(orgID, runID))
}

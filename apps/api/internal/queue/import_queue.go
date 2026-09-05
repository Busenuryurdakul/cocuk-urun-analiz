package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
)

const ImportQueueKey = "marketplace:import:queue"

type ImportJob struct {
	ImportRunID    string `json:"importRunId"`
	OrganizationID string `json:"organizationId"`
}

type ImportQueue struct {
	Redis *redis.Client
}

func (q *ImportQueue) Enqueue(ctx context.Context, job ImportJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.Redis.LPush(ctx, ImportQueueKey, string(payload))
}

func (q *ImportQueue) Dequeue(ctx context.Context, timeout time.Duration) (*ImportJob, error) {
	raw, err := q.Redis.BRPop(ctx, ImportQueueKey, timeout)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var job ImportJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

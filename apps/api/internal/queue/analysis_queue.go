package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
)

const AnalysisQueueKey = "analysis:run:queue"

type AnalysisJob struct {
	AnalysisRunID  string `json:"analysisRunId"`
	OrganizationID string `json:"organizationId"`
}

type AnalysisQueue struct {
	Redis *redis.Client
}

func (q *AnalysisQueue) Enqueue(ctx context.Context, job AnalysisJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.Redis.LPush(ctx, AnalysisQueueKey, string(payload))
}

func (q *AnalysisQueue) Dequeue(ctx context.Context, timeout time.Duration) (*AnalysisJob, error) {
	raw, err := q.Redis.BRPop(ctx, AnalysisQueueKey, timeout)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var job AnalysisJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

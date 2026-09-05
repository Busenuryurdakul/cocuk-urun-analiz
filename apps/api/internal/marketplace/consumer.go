package marketplace

import (
	"context"
	"log"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/queue"
)

type Consumer struct {
	Service  *Service
	Interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewConsumer(svc *Service, interval time.Duration) *Consumer {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Consumer{
		Service:  svc,
		Interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (c *Consumer) Start() {
	go c.loop()
}

func (c *Consumer) loop() {
	defer close(c.doneCh)
	for {
		select {
		case <-c.stopCh:
			return
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			job, err := c.Service.Queue.Dequeue(ctx, c.Interval)
			cancel()
			if err != nil {
				log.Printf("import consumer dequeue: %v", err)
				time.Sleep(c.Interval)
				continue
			}
			if job == nil {
				continue
			}
			c.processJob(*job)
		}
	}
}

func (c *Consumer) processJob(job queue.ImportJob) {
	orgID, runID, err := parseJobIDs(job)
	if err != nil {
		log.Printf("import consumer invalid job: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := c.Service.ProcessRun(ctx, orgID, runID); err != nil {
		log.Printf("import consumer process run %s: %v", job.ImportRunID, err)
	}
}

func (c *Consumer) Stop(ctx context.Context) {
	close(c.stopCh)
	select {
	case <-c.doneCh:
	case <-ctx.Done():
	}
}

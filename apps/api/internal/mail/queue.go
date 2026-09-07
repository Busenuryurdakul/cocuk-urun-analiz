package mail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const mailQueueKey = "mail:queue"

type QueueRedis interface {
	LPush(ctx context.Context, key, value string) error
	BRPop(ctx context.Context, key string, timeout time.Duration) (string, error)
}

type queuedMessage struct {
	To         string `json:"to"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	HTMLBody   string `json:"htmlBody"`
	Attempts   int    `json:"attempts"`
	MaxRetries int    `json:"maxRetries"`
}

type QueuedService struct {
	inner      Service
	redis      QueueRedis
	enabled    bool
	maxRetries int
}

func NewQueuedService(inner Service, redis QueueRedis, enabled bool, maxRetries int) *QueuedService {
	if maxRetries < 1 {
		maxRetries = 3
	}
	return &QueuedService{
		inner:      inner,
		redis:      redis,
		enabled:    enabled,
		maxRetries: maxRetries,
	}
}

// SendImmediate delivers through the inner provider, bypassing the queue.
// Use for auth-critical mail so delivery failures surface to the caller.
func (q *QueuedService) SendImmediate(ctx context.Context, msg Message) error {
	return q.inner.Send(ctx, msg)
}

func (q *QueuedService) Send(ctx context.Context, msg Message) error {
	if !q.enabled || q.redis == nil {
		return q.inner.Send(ctx, msg)
	}
	raw, err := json.Marshal(queuedMessage{
		To:         msg.To,
		Subject:    msg.Subject,
		Body:       msg.Body,
		HTMLBody:   msg.HTMLBody,
		Attempts:   0,
		MaxRetries: q.maxRetries,
	})
	if err != nil {
		return err
	}
	return q.redis.LPush(ctx, mailQueueKey, string(raw))
}

func (q *QueuedService) StartWorker(ctx context.Context, interval time.Duration) {
	if !q.enabled || q.redis == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			raw, err := q.redis.BRPop(ctx, mailQueueKey, interval)
			if err != nil {
				log.Printf("mail queue pop: %v", err)
				continue
			}
			if raw == "" {
				continue
			}
			var item queuedMessage
			if err := json.Unmarshal([]byte(raw), &item); err != nil {
				log.Printf("mail queue decode: %v", err)
				continue
			}
			msg := Message{To: item.To, Subject: item.Subject, Body: item.Body, HTMLBody: item.HTMLBody}
			if err := q.inner.Send(ctx, msg); err != nil {
				item.Attempts++
				if item.Attempts < item.MaxRetries {
					requeue, _ := json.Marshal(item)
					_ = q.redis.LPush(ctx, mailQueueKey, string(requeue))
					time.Sleep(time.Duration(item.Attempts) * time.Second)
				} else {
					log.Printf("mail delivery failed after %d attempts to %s: %v", item.Attempts, item.To, err)
				}
			} else {
				log.Printf("mail sent to %s subject=%q", item.To, item.Subject)
			}
		}
	}()
}

func (q *QueuedService) Inner() Service {
	return q.inner
}

func (q *QueuedService) QueueKey() string {
	return mailQueueKey
}

func (q *QueuedService) String() string {
	if q.enabled {
		return fmt.Sprintf("queued-mail(%s)", mailQueueKey)
	}
	return "direct-mail"
}

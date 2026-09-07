package mail

import "context"

type Message struct {
	To       string
	Subject  string
	Body     string
	HTMLBody string
}

type Service interface {
	Send(ctx context.Context, msg Message) error
}

type immediateSender interface {
	SendImmediate(context.Context, Message) error
}

// DeliverNow sends through the provider immediately, skipping the Redis queue.
func DeliverNow(ctx context.Context, s Service, msg Message) error {
	if s == nil {
		return nil
	}
	if immediate, ok := s.(immediateSender); ok {
		return immediate.SendImmediate(ctx, msg)
	}
	return s.Send(ctx, msg)
}

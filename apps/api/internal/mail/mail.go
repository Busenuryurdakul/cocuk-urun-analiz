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

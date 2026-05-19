package provider

import "context"

type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

type EmailMessage struct {
	To      string
	Subject string
	Body    string
}

package provider

import "context"

// EmailSender is the port for the notification provider.
// Business logic depends only on this interface, never on concrete providers.
// This allows swapping SMTP ↔ Mailjet ↔ Mock without touching domain code.
type EmailSender interface {
	// Send dispatches a notification email.
	// Returns an error if the delivery attempt fails (triggering retry logic).
	Send(ctx context.Context, msg EmailMessage) error
}

// EmailMessage carries all data needed to send a notification.
type EmailMessage struct {
	To      string
	Subject string
	Body    string
}

package ports

import (
	"context"
	"expense-tracking/internal/core/domain"
)

// EmailProvider defines the interface for interacting with the email provider
type EmailProvider interface {
	// FetchUnreadBankEmails retrieves unread emails from the specific bank
	FetchUnreadBankEmails(ctx context.Context, senderQuery string) ([]domain.RawEmail, error)

	// MarkAsProcessed marks an email as read and applies a custom label
	MarkAsProcessed(ctx context.Context, messageID string, labelName string) error

	// SearchEmails allows fetching emails with a custom query (e.g. for backfilling)
	SearchEmails(ctx context.Context, query string) ([]domain.RawEmail, error)
}

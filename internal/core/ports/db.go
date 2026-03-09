package ports

import (
	"context"
	"expense-tracking/internal/core/domain"
)

// StateManager defines the interface for tracking processed items.
type StateManager interface {
	// Start starts the state manager and runs any necessary migrations
	Start(ctx context.Context) error
	
	// ExistsMessageID checks if a message has already been processed successfully
	ExistsMessageID(ctx context.Context, messageID string) (bool, error)
	
	// SaveMessageState records the processing status of a message
	SaveMessageState(ctx context.Context, messageID string, status string) error

	// SaveExpense stores the extracted expense details in the database
	SaveExpense(ctx context.Context, messageID string, expense domain.Expense) error

	// ListExpenses returns the most recent expenses
	ListExpenses(ctx context.Context, limit int) ([]domain.Expense, error)

	// Close closes the underlying connection
	Close()
}

package db

import (
	"context"
	"fmt"
	"time"

	"expense-tracking/internal/core/domain"
	"expense-tracking/internal/core/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStateManager struct {
	pool *pgxpool.Pool
}

// NewPostgresStateManager creates a new PostgreSQL state manager using pgxpool.
func NewPostgresStateManager(ctx context.Context, connString string) (ports.StateManager, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &postgresStateManager{
		pool: pool,
	}, nil
}

func (m *postgresStateManager) Start(ctx context.Context) error {
	// Schema migration for tracking message IDs and expense records
	query := `
		CREATE TABLE IF NOT EXISTS processed_messages (
			id SERIAL PRIMARY KEY,
			message_id VARCHAR(255) UNIQUE NOT NULL,
			status VARCHAR(50) NOT NULL,
			process_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_message_id ON processed_messages(message_id);

		CREATE TABLE IF NOT EXISTS expenses (
			id SERIAL PRIMARY KEY,
			message_id VARCHAR(255) UNIQUE NOT NULL,
			date TIMESTAMP WITH TIME ZONE NOT NULL,
			merchant VARCHAR(255) NOT NULL,
			amount DECIMAL(15, 2) NOT NULL,
			currency VARCHAR(10) NOT NULL,
			category VARCHAR(100),
			bank VARCHAR(50),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_expense_date ON expenses(date);
		CREATE INDEX IF NOT EXISTS idx_expense_message_id ON expenses(message_id);
	`
	_, err := m.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func (m *postgresStateManager) ExistsMessageID(ctx context.Context, messageID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM processed_messages WHERE message_id = $1 AND status = 'SUCCESS')`
	err := m.pool.QueryRow(ctx, query, messageID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check message ID existence: %w", err)
	}
	return exists, nil
}

func (m *postgresStateManager) SaveMessageState(ctx context.Context, messageID string, status string) error {
	query := `
		INSERT INTO processed_messages (message_id, status, process_date)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id) 
		DO UPDATE SET status = EXCLUDED.status, process_date = EXCLUDED.process_date;
	`
	_, err := m.pool.Exec(ctx, query, messageID, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save message state: %w", err)
	}
	return nil
}

func (m *postgresStateManager) SaveExpense(ctx context.Context, messageID string, expense domain.Expense) error {
	query := `
		INSERT INTO expenses (message_id, date, merchant, amount, currency, category, bank)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (message_id) DO NOTHING;
	`
	_, err := m.pool.Exec(ctx, query,
		messageID,
		expense.Date,
		expense.Merchant,
		expense.Amount,
		expense.Currency,
		expense.Category,
		expense.Bank,
	)
	if err != nil {
		return fmt.Errorf("failed to save expense: %w", err)
	}
	return nil
}

func (m *postgresStateManager) Close() {
	if m.pool != nil {
		m.pool.Close()
	}
}

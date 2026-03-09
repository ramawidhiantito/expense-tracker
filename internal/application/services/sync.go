package services

import (
	"context"
	"fmt"
	"log"

	"expense-tracking/internal/core/domain"
	"expense-tracking/internal/core/ports"
)

type SyncService struct {
	emailProvider ports.EmailProvider
	sheetProvider ports.SheetProvider
	stateManager  ports.StateManager
	parsers       []ports.Parser
	sheetID       string
	sheetRange    string
}

// NewSyncService creates the main orchestrator for tracking expenses
func NewSyncService(
	email ports.EmailProvider,
	sheet ports.SheetProvider,
	state ports.StateManager,
	parsers []ports.Parser,
	sheetID string,
	sheetRange string,
) *SyncService {
	return &SyncService{
		emailProvider: email,
		sheetProvider: sheet,
		stateManager:  state,
		parsers:       parsers,
		sheetID:       sheetID,
		sheetRange:    sheetRange,
	}
}

// Run executes a single iteration of the sync pipeline
// If unreadOnly is true, it only fetches unread emails. If false, it searches all emails matching the query.
func (s *SyncService) Run(ctx context.Context, query string, unreadOnly bool) (int, error) {
	var emails []domain.RawEmail
	var err error

	log.Printf("Fetching emails matching query: %s\n", query)
	if unreadOnly {
		emails, err = s.emailProvider.FetchUnreadBankEmails(ctx, query)
	} else {
		// Used for backfill
		emails, err = s.emailProvider.SearchEmails(ctx, query)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to fetch emails: %w", err)
	}

	log.Printf("Found %d unread emails.\n", len(emails))
	processedCount := 0

	for _, email := range emails {
		// 1. Check idempotency
		exists, err := s.stateManager.ExistsMessageID(ctx, email.ID)
		if err != nil {
			log.Printf("WARN: Failed to check state for msg %s: %v\n", email.ID, err)
			continue
		}
		if exists {
			log.Printf("Message %s already processed, marking as read.\n", email.ID)
			_ = s.emailProvider.MarkAsProcessed(ctx, email.ID, "EXPENSE_TRACKED")
			continue
		}

		// 2. Find suitable parser
		var activeParser ports.Parser
		for _, p := range s.parsers {
			if p.CanParse(email) {
				activeParser = p
				break
			}
		}

		if activeParser == nil {
			log.Printf("No parser found for email subject: %s\n", email.Subject)
			// Save state as SKIPPED so we don't try again unless backfilling
			_ = s.stateManager.SaveMessageState(ctx, email.ID, "SKIPPED")
			continue
		}

		// 3. Parse
		expense, err := activeParser.Parse(email)
		if err != nil {
			log.Printf("Failed to parse email %s: %v\n", email.ID, err)
			_ = s.stateManager.SaveMessageState(ctx, email.ID, "FAILED")
			continue
		}

		// 4. Export to Sheets and DB
		err = s.sheetProvider.AppendExpense(ctx, s.sheetID, s.sheetRange, *expense)
		if err != nil {
			log.Printf("Failed to export expense to sheets (msg %s): %v\n", email.ID, err)
			// Do not save state as FAILED here, we want it to retry on the next run
			continue
		}

		err = s.stateManager.SaveExpense(ctx, email.ID, *expense)
		if err != nil {
			log.Printf("Failed to save expense to database (msg %s): %v\n", email.ID, err)
			// Decide if this should be fatal for the message.
			// Since it's already in Sheets, we might want to continue or retry.
			// For now, let's just log and continue, or we can consider it a partial failure.
		}

		// 5. Mark as processed in DB and Gmail
		err = s.stateManager.SaveMessageState(ctx, email.ID, "SUCCESS")
		if err != nil {
			log.Printf("WARN: Failed to save SUCCESS state for %s: %v\n", email.ID, err)
		}

		err = s.emailProvider.MarkAsProcessed(ctx, email.ID, "EXPENSE_TRACKED")
		if err != nil {
			log.Printf("WARN: Failed to mark email %s as processed in Gmail: %v\n", email.ID, err)
		}

		log.Printf("Successfully tracked expense: %s - %.2f\n", expense.Merchant, expense.Amount)
		processedCount++
	}

	return processedCount, nil
}

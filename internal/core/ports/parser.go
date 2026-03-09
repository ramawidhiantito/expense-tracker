package ports

import "expense-tracking/internal/core/domain"

// Parser defines the interface for extracting structured expense data from emails
type Parser interface {
	// Parse evaluates the raw email body and extracts the expense data
	Parse(email domain.RawEmail) (*domain.Expense, error)

	// CanParse returns true if this parser is capable of handling the specific email (e.g., checking subject/sender)
	CanParse(email domain.RawEmail) bool
}

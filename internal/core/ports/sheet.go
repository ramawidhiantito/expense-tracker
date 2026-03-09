package ports

import (
	"context"
	"expense-tracking/internal/core/domain"
)

// SheetProvider defines the interface for exporting data to a spreadsheet
type SheetProvider interface {
	// AppendExpense adds a single expense record to the sheet
	AppendExpense(ctx context.Context, spreadsheetID string, sheetRange string, expense domain.Expense) error
}

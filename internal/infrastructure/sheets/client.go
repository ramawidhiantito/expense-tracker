package sheets

import (
	"context"
	"fmt"
	"os"

	"expense-tracking/internal/core/domain"
	"expense-tracking/internal/core/ports"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type sheetsProvider struct {
	srv *sheets.Service
}

// NewSheetsProvider initializes a Sheets client with a Service Account JSON file.
func NewSheetsProvider(ctx context.Context, serviceAccountFile string) (ports.SheetProvider, error) {
	b, err := os.ReadFile(serviceAccountFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read service account key file: %v", err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(b), option.WithScopes(sheets.SpreadsheetsScope))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	return &sheetsProvider{srv: srv}, nil
}

func (s *sheetsProvider) AppendExpense(ctx context.Context, spreadsheetID string, sheetRange string, expense domain.Expense) error {
	// Format the row data according to the design:
	// A: Date, B: Merchant, C: Amount, D: Category, E: Bank
	dateStr := expense.Date.Format("2006-01-02 15:04")

	row := []interface{}{
		dateStr,
		expense.Merchant,
		expense.Amount,
		expense.Category,
		expense.Bank,
	}

	vr := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}

	// USER_ENTERED allows sheets to detect the amount as a number and date as a date
	_, err := s.srv.Spreadsheets.Values.Append(spreadsheetID, sheetRange, vr).
		ValueInputOption("USER_ENTERED").
		Do()

	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	return nil
}

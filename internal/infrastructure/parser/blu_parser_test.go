package parser

import (
	"expense-tracking/internal/core/domain"
	"os"
	"testing"
)

func TestBluParser(t *testing.T) {
	// Path relative to this file
	htmlContent, err := os.ReadFile("../../../cmd/tracker/blu.html")
	if err != nil {
		t.Fatalf("Failed to read blu.html: %v", err)
	}

	p := NewBluParser()
	email := domain.RawEmail{
		Sender:  "blu <receipts@blubybcadigital.id>",
		Subject: "Transaction Notification",
		Body:    string(htmlContent),
	}

	if !p.CanParse(email) {
		t.Errorf("Expected CanParse to be true")
	}

	expense, err := p.Parse(email)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if expense.Merchant != "JAKLINGKO JAKARTA PUSAT" {
		t.Errorf("Expected merchant 'JAKLINGKO JAKARTA PUSAT', got '%s'", expense.Merchant)
	}

	if expense.Amount != 5750.00 {
		t.Errorf("Expected amount 5750.00, got %f", expense.Amount)
	}

	if expense.Currency != "Rupiah" {
		t.Errorf("Expected currency 'Rupiah', got '%s'", expense.Currency)
	}

	if expense.Category != "QRIS" {
		t.Errorf("Expected category 'QRIS', got '%s'", expense.Category)
	}

	expectedDate := "2026-02-24 07:28:49"
	if expense.Date.Format("2006-01-02 15:04:05") != expectedDate {
		t.Errorf("Expected date %s, got %s", expectedDate, expense.Date.Format("2006-01-02 15:04:05"))
	}
}

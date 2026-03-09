package parser

import (
	"expense-tracking/internal/core/domain"
	"os"
	"testing"
)

func TestBCAParser_TransferBCA(t *testing.T) {
	htmlContent, err := os.ReadFile("../../../cmd/tracker/bca4.html")
	if err != nil {
		t.Fatalf("Failed to read bca4.html: %v", err)
	}

	p := NewBCAParser()
	email := domain.RawEmail{
		Sender: "BCA <bca@bca.co.id>",
		Body:   string(htmlContent),
	}

	expense, err := p.Parse(email)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if expense.Merchant != "RAMA WIDHIANTITO" {
		t.Errorf("Expected merchant 'RAMA WIDHIANTITO', got '%s'", expense.Merchant)
	}

	if expense.Amount != 500000.00 {
		t.Errorf("Expected amount 500000.00, got %f", expense.Amount)
	}

	if expense.Category != "Transfer to BCA Account" {
		t.Errorf("Expected category 'Transfer to BCA Account', got '%s'", expense.Category)
	}
}

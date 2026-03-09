package parser

import (
	"expense-tracking/internal/core/domain"
	"os"
	"testing"
)

func TestBCAParser(t *testing.T) {
	// Path relative to this file
	htmlContent, err := os.ReadFile("../../../cmd/tracker/bca.html")
	if err != nil {
		t.Fatalf("Failed to read bca.html: %v", err)
	}

	p := NewBCAParser()
	email := domain.RawEmail{
		Sender:  "BCA <bca@bca.co.id>",
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

	if expense.Merchant != "PT VISIONET INTERNASIONAL / OVO" {
		t.Errorf("Expected merchant 'PT VISIONET INTERNASIONAL / OVO', got '%s'", expense.Merchant)
	}

	if expense.Amount != 20000.00 {
		t.Errorf("Expected amount 20000.00, got %f", expense.Amount)
	}

	if expense.Currency != "Rupiah" {
		t.Errorf("Expected currency 'Rupiah', got '%s'", expense.Currency)
	}

	if expense.Category != "Transfer to BCA Virtual Account" {
		t.Errorf("Expected category 'Transfer to BCA Virtual Account', got '%s'", expense.Category)
	}

	expectedDate := "2026-03-01 17:42:17"
	if expense.Date.Format("2006-01-02 15:04:05") != expectedDate {
		t.Errorf("Expected date %s, got %s", expectedDate, expense.Date.Format("2006-01-02 15:04:05"))
	}
}

func TestBCAParser_QRIS(t *testing.T) {
	// Path relative to this file
	htmlContent, err := os.ReadFile("../../../cmd/tracker/bca2.html")
	if err != nil {
		t.Fatalf("Failed to read bca2.html: %v", err)
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

	if expense.Merchant != "Gantominang Uin, Pisangan" {
		t.Errorf("Expected merchant 'Gantominang Uin, Pisangan', got '%s'", expense.Merchant)
	}

	if expense.Amount != 20000.00 {
		t.Errorf("Expected amount 20000.00, got %f", expense.Amount)
	}

	if expense.Category != "QRIS Payment" {
		t.Errorf("Expected category 'QRIS Payment', got '%s'", expense.Category)
	}

	expectedDate := "2026-03-02 17:54:57"
	if expense.Date.Format("2006-01-02 15:04:05") != expectedDate {
		t.Errorf("Expected date %s, got %s", expectedDate, expense.Date.Format("2006-01-02 15:04:05"))
	}
}

func TestBCAParser_Transfer(t *testing.T) {
	htmlContent, err := os.ReadFile("../../../cmd/tracker/bca3.html")
	if err != nil {
		t.Fatalf("Failed to read bca3.html: %v", err)
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

	if expense.Merchant != "RAMA WIDHIANTITO - BSI" {
		t.Errorf("Expected merchant 'RAMA WIDHIANTITO - BSI', got '%s'", expense.Merchant)
	}

	if expense.Amount != 7250000.00 {
		t.Errorf("Expected amount 7250000.00, got %f", expense.Amount)
	}

	if expense.Category != "Transfer to BSI" {
		t.Errorf("Expected category 'Transfer to BSI', got '%s'", expense.Category)
	}

	if expense.Bank != "BCA" {
		t.Errorf("Expected bank 'BCA', got '%s'", expense.Bank)
	}

	expectedDate := "2026-02-25 05:00:46"
	if expense.Date.Format("2006-01-02 15:04:05") != expectedDate {
		t.Errorf("Expected date %s, got %s", expectedDate, expense.Date.Format("2006-01-02 15:04:05"))
	}
}

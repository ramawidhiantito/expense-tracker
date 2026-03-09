package domain

import "time"

// Expense represents the final extracted expense record
type Expense struct {
	Date     time.Time
	Merchant string
	Amount   float64
	Currency string
	Category string
	Bank     string
}

// RawEmail represents the raw email fetched from the provider
type RawEmail struct {
	ID      string
	Date    time.Time
	Sender  string
	Subject string
	Body    string // Could be plaintext or decoded HTML
}

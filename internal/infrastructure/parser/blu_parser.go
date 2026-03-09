package parser

import (
	"expense-tracking/internal/core/domain"
	"expense-tracking/internal/core/ports"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type bluParser struct {
}

// NewBluParser creates a dedicated parser for Blu BCA Digital emails
func NewBluParser() ports.Parser {
	return &bluParser{}
}

func (p *bluParser) CanParse(email domain.RawEmail) bool {
	// receipts@blubybcadigital.id
	return strings.Contains(strings.ToLower(email.Sender), "receipts@blubybcadigital.id")
}

func (p *bluParser) Parse(email domain.RawEmail) (*domain.Expense, error) {
	expense := &domain.Expense{
		Currency: "Rupiah",
		Bank:     "BLU",
	}

	// Prepare clean body text
	bodyText := email.Body
	bodyText = html.UnescapeString(bodyText)
	// Strip HTML tags
	bodyText = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(bodyText, " ")
	// Normalize whitespace
	bodyText = regexp.MustCompile(`\s+`).ReplaceAllString(bodyText, " ")

	// Extract Category (Tipe Transaksi)
	categoryRegex := regexp.MustCompile(`(?i)Tipe Transaksi\s+(.+?)\s+(?:No\. Ref|Cheers)`)
	if matches := categoryRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		expense.Category = strings.TrimSpace(matches[1])
	}

	// Extract Merchant
	// Merchant is between "bluAccount" and "Nominal Tagihan"
	// Example: "... Rama Widhiantito bluAccount JAKLINGKO JAKARTA PUSAT Nominal Tagihan ..."
	merchantRegex := regexp.MustCompile(`(?i)bluAccount\s+(.+?)\s+Nominal Tagihan`)
	if matches := merchantRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		expense.Merchant = strings.TrimSpace(matches[1])
	}

	// Extract Amount (Nominal Tagihan)
	amountRegex := regexp.MustCompile(`(?i)Nominal Tagihan\s+(?:IDR|Rp)?\s*([0-9.,]+)`)
	if matches := amountRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		amtStr := matches[1]
		lastComma := strings.LastIndex(amtStr, ",")
		lastDot := strings.LastIndex(amtStr, ".")

		if lastComma > lastDot {
			// Indonesian format: 5.750,00
			amtStr = strings.ReplaceAll(amtStr, ".", "")
			amtStr = strings.ReplaceAll(amtStr, ",", ".")
		} else if lastDot > lastComma {
			// US format: 5,750.00
			amtStr = strings.ReplaceAll(amtStr, ",", "")
		} else if lastComma != -1 {
			amtStr = strings.ReplaceAll(amtStr, ",", ".")
		}

		if val, err := strconv.ParseFloat(amtStr, 64); err == nil {
			expense.Amount = val
		} else {
			return nil, fmt.Errorf("failed to parse blu amount '%s': %v", matches[1], err)
		}
	} else {
		return nil, fmt.Errorf("blu amount (Nominal Tagihan) not found")
	}

	// Extract Date (Tgl & Jam Transaksi)
	// 24 Feb 2026 07:28:49 WIB
	dateRegex := regexp.MustCompile(`(?i)Tgl & Jam Transaksi\s+([0-9]{2} [a-zA-Z]{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2})`)
	if matches := dateRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		// Use "02 Jan 2006 15:04:05" layout
		if t, err := time.Parse("02 Jan 2006 15:04:05", strings.TrimSpace(matches[1])); err == nil {
			expense.Date = t
		} else {
			expense.Date = email.Date
		}
	} else {
		expense.Date = email.Date
	}

	return expense, nil
}

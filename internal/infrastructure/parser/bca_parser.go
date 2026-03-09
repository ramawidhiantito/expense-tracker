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

type bcaParser struct {
}

// NewBCAParser creates a dedicated parser for BCA emails
func NewBCAParser() ports.Parser {
	return &bcaParser{}
}

func (p *bcaParser) CanParse(email domain.RawEmail) bool {
	// BCA <bca@bca.co.id>
	return strings.Contains(strings.ToLower(email.Sender), "bca@bca.co.id")
}

func (p *bcaParser) Parse(email domain.RawEmail) (*domain.Expense, error) {
	expense := &domain.Expense{
		Currency: "Rupiah",
		Bank:     "BCA",
	}

	// Prepare clean body text
	bodyText := email.Body
	bodyText = html.UnescapeString(bodyText)
	// Strip HTML tags
	bodyText = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(bodyText, " ")
	// Normalize whitespace
	bodyText = regexp.MustCompile(`\s+`).ReplaceAllString(bodyText, " ")

	// Extract Category (Transaction Type or Transfer Type)
	categoryRegex := regexp.MustCompile(`(?i)(?:Transaction Type|Transfer Type)\s*:\s*(.+?)\s*(?:Payment to|Source of Fund|Transaction Date|Status)`)
	if matches := categoryRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		expense.Category = strings.TrimSpace(matches[1])
	}

	// Extract Merchant (Company/Product Name or Payment to)
	merchantRegex := regexp.MustCompile(`(?i)(?:Company/Product Name|Payment to)\s*:\s*(.+?)\s*(?:Merchant Location|Pay Amount|Total Payment|Bill Total|billdesc|Description)`)
	if matches := merchantRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		expense.Merchant = strings.TrimSpace(matches[1])
	}

	// Fallback: Beneficiary Name + Beneficiary Bank
	if expense.Merchant == "" {
		nameRegex := regexp.MustCompile(`(?i)Beneficiary Name\s*:\s*(.+?)\s*(?:Beneficiary Bank|Beneficiary Account|Amount|Transfer Amount|Remarks)`)
		bankRegex := regexp.MustCompile(`(?i)Beneficiary Bank\s*:\s*(.+?)\s*(?:Beneficiary Account|Amount|Transfer Amount|Fee)`)
		var name, bank string
		if matches := nameRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
			name = strings.TrimSpace(matches[1])
		}
		if matches := bankRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
			bank = strings.TrimSpace(matches[1])
		}
		if name != "" && bank != "" {
			expense.Merchant = name + " - " + bank
		} else if name != "" {
			expense.Merchant = name
		}
	}

	// Extract Amount (Total Payment or Amount or Transfer Amount)
	amountRegex := regexp.MustCompile(`(?i)(?:Total Payment|Transfer Amount|Amount)\s*:\s*(?:IDR|Rp)?\s*([0-9.,]+)`)
	if matches := amountRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		amtStr := matches[1]
		lastComma := strings.LastIndex(amtStr, ",")
		lastDot := strings.LastIndex(amtStr, ".")

		if lastComma > lastDot {
			// Indonesian format: 20.000,00
			amtStr = strings.ReplaceAll(amtStr, ".", "")
			amtStr = strings.ReplaceAll(amtStr, ",", ".")
		} else if lastDot > lastComma {
			// US format: 20,000.00
			amtStr = strings.ReplaceAll(amtStr, ",", "")
		} else if lastComma != -1 {
			amtStr = strings.ReplaceAll(amtStr, ",", ".")
		}
		// Final parse
		if val, err := strconv.ParseFloat(amtStr, 64); err == nil {
			expense.Amount = val
		} else {
			return nil, fmt.Errorf("failed to parse BCA amount '%s': %v", matches[1], err)
		}
	} else {
		return nil, fmt.Errorf("BCA amount (Total Payment/Amount) not found")
	}

	// Extract Date (Transaction Date)
	dateRegex := regexp.MustCompile(`(?i)Transaction Date\s*:\s*([0-9]{2} [a-zA-Z]{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2})`)
	if matches := dateRegex.FindStringSubmatch(bodyText); len(matches) > 1 {
		if t, err := time.Parse("02 Jan 2006 15:04:05", strings.TrimSpace(matches[1])); err == nil {
			expense.Date = t
		} else {
			expense.Date = email.Date // Fallback
		}
	} else {
		expense.Date = email.Date // Fallback
	}

	return expense, nil
}

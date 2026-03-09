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

type RegexConfig struct {
	BankName         string
	SenderRegex      *regexp.Regexp
	SubjectRegex     *regexp.Regexp
	AmountRegex      *regexp.Regexp // Capture group 1 for the amount string
	DateRegex        *regexp.Regexp // Capture group 1 for the date string
	DateFormat       string         // layout for time.Parse
	MerchantRegex    *regexp.Regexp // Capture group 1 for merchant
	ReferenceIDRegex *regexp.Regexp // Capture group 1 for Ref ID
}

type regexParser struct {
	config RegexConfig
}

// NewRegexParser creates a new customizable regex-based parser
func NewRegexParser(config RegexConfig) ports.Parser {
	return &regexParser{config: config}
}

func (p *regexParser) CanParse(email domain.RawEmail) bool {
	if p.config.SenderRegex != nil && !p.config.SenderRegex.MatchString(email.Sender) {
		return false
	}
	if p.config.SubjectRegex != nil && !p.config.SubjectRegex.MatchString(email.Subject) {
		return false
	}
	return true
}

func (p *regexParser) Parse(email domain.RawEmail) (*domain.Expense, error) {
	expense := &domain.Expense{
		Currency: "IDR", // Default
	}

	// Prepare a clean body text for parsing
	bodyText := email.Body
	// Unescape HTML entities (e.g., &amp; -> &, &nbsp; -> space)
	bodyText = html.UnescapeString(bodyText)
	// Strip HTML tags and replace with space to avoid merging words
	bodyText = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(bodyText, " ")
	// Normalize all whitespace (including newlines) to a single space
	bodyText = regexp.MustCompile(`\s+`).ReplaceAllString(bodyText, " ")

	// Extract Amount
	if p.config.AmountRegex != nil {
		matches := p.config.AmountRegex.FindStringSubmatch(bodyText)
		if len(matches) > 1 {
			// clean amount string: remove thousand separators (.) and replace decimal separator (,)
			amtStr := strings.ReplaceAll(matches[1], ".", "")
			amtStr = strings.ReplaceAll(amtStr, ",", ".")
			if val, err := strconv.ParseFloat(amtStr, 64); err == nil {
				expense.Amount = val
			} else {
				return nil, fmt.Errorf("failed to parse amount '%s': %v", matches[1], err)
			}
		} else {
			return nil, fmt.Errorf("amount not found in email using regex pattern")
		}
	}

	// Extract Date
	if p.config.DateRegex != nil {
		matches := p.config.DateRegex.FindStringSubmatch(bodyText)
		if len(matches) > 1 {
			if t, err := time.Parse(p.config.DateFormat, strings.TrimSpace(matches[1])); err == nil {
				expense.Date = t
			} else {
				return nil, fmt.Errorf("failed to parse date '%s': %v", matches[1], err)
			}
		} else {
			expense.Date = email.Date // Fallback to email sent date
		}
	} else {
		expense.Date = email.Date
	}

	// Extract Merchant
	if p.config.MerchantRegex != nil {
		matches := p.config.MerchantRegex.FindStringSubmatch(bodyText)
		if len(matches) > 1 {
			expense.Merchant = strings.TrimSpace(matches[1])
		}
	}

	return expense, nil
}

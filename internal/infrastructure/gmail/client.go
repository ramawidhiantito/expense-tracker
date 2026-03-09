package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"expense-tracking/internal/core/domain"
	"expense-tracking/internal/core/ports"

	"google.golang.org/api/gmail/v1"
)

type gmailProvider struct {
	srv *gmail.Service
}

func NewGmailProvider(srv *gmail.Service) ports.EmailProvider {
	return &gmailProvider{srv: srv}
}

func (g *gmailProvider) FetchUnreadBankEmails(ctx context.Context, senderQuery string) ([]domain.RawEmail, error) {
	// Query explicitly for unread messages matching the sender
	query := fmt.Sprintf("is:unread %s", senderQuery)
	return g.SearchEmails(ctx, query)
}

func (g *gmailProvider) SearchEmails(ctx context.Context, query string) ([]domain.RawEmail, error) {
	// Retrieve messages matching the query
	msgList, err := g.srv.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	var rawEmails []domain.RawEmail
	for _, m := range msgList.Messages {
		msg, err := g.srv.Users.Messages.Get("me", m.Id).Format("full").Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get message %s: %w", m.Id, err)
		}

		var subject, sender string
		for _, header := range msg.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
			}
			if header.Name == "From" {
				sender = header.Value
			}
			if subject != "" && sender != "" {
				break
			}
		}

		body := decodeEmailBody(msg.Payload)
		if body == "" {
			body = msg.Snippet
		}

		// Parse internal date (which is in ms)
		internalDate := time.UnixMilli(msg.InternalDate)

		rawEmails = append(rawEmails, domain.RawEmail{
			ID:      m.Id,
			Date:    internalDate,
			Sender:  sender,
			Subject: subject,
			Body:    body,
		})
	}

	return rawEmails, nil
}

func (g *gmailProvider) MarkAsProcessed(ctx context.Context, messageID string, labelName string) error {
	// Find or create the label
	labelID, err := g.getOrCreateLabel(labelName)
	if err != nil {
		return fmt.Errorf("failed to get/create label %s: %w", labelName, err)
	}

	// Remove UNREAD label, add custom label
	req := &gmail.ModifyMessageRequest{
		AddLabelIds:    []string{labelID},
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err = g.srv.Users.Messages.Modify("me", messageID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to modify message labels: %w", err)
	}

	return nil
}

// decodeEmailBody attempts to extract the plain text or HTML body from the payload
func decodeEmailBody(payload *gmail.MessagePart) string {
	if payload == nil {
		return ""
	}

	var data string

	// Check if this part itself is the body
	if payload.MimeType == "text/plain" || payload.MimeType == "text/html" {
		data = payload.Body.Data
	}

	// Recursively search parts
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" {
			data = part.Body.Data
			break // Prefer plain text
		}
		if part.MimeType == "text/html" && data == "" {
			data = part.Body.Data
		}
		// If it's multipart (e.g., multipart/alternative), recurse
		if strings.HasPrefix(part.MimeType, "multipart/") {
			result := decodeEmailBody(part)
			if result != "" {
				data = result
				break
			}
		}
	}

	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return data
	}
	return string(decoded)
}

func (g *gmailProvider) getOrCreateLabel(labelName string) (string, error) {
	listRes, err := g.srv.Users.Labels.List("me").Do()
	if err != nil {
		return "", err
	}

	for _, l := range listRes.Labels {
		if l.Name == labelName {
			return l.Id, nil
		}
	}

	// Label doesn't exist, create it
	newLabel := &gmail.Label{
		Name:                  labelName,
		LabelListVisibility:   "labelShow",
		MessageListVisibility: "show",
	}
	createdLabel, err := g.srv.Users.Labels.Create("me", newLabel).Do()
	if err != nil {
		return "", err
	}

	return createdLabel.Id, nil
}

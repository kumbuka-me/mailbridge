// Package application contains transport-independent mail forwarding use cases.
package application

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"uuid"
)

// ErrInvalidRequest identifies a mail request that cannot be forwarded.
var ErrInvalidRequest = errors.New("invalid mail request")

// BodyFormat identifies the MIME body representation used for an email.
type BodyFormat string

const (
	// BodyFormatText sends the message body as text/plain.
	BodyFormatText BodyFormat = "text"
	// BodyFormatHTML sends the message body as text/html.
	BodyFormatHTML BodyFormat = "html"
)

// Recipient identifies one message recipient.
type Recipient struct {
	// Email is the recipient mailbox.
	Email string `json:"email"`
	// DisplayName is the optional user-facing recipient name.
	DisplayName string `json:"display_name,omitempty"`
}

// Recipients groups the RFC recipient classes for one email message.
type Recipients struct {
	// To contains primary recipients visible in the To header.
	To []Recipient `json:"to"`
	// CC contains carbon-copy recipients visible in the Cc header.
	CC []Recipient `json:"cc"`
	// BCC contains blind-carbon-copy recipients omitted from message headers.
	BCC []Recipient `json:"bcc"`
}

// Content contains the subject and body of one email message.
type Content struct {
	// Subject is used as the email subject.
	Subject string `json:"subject"`
	// Body is the email body.
	Body string `json:"body"`
	// BodyFormat optionally overrides the configured default body format.
	BodyFormat BodyFormat `json:"body_format,omitempty"`
}

// Request describes exactly one email message.
type Request struct {
	// Recipients contains the To, Cc, and Bcc recipients for the message.
	Recipients Recipients `json:"recipients"`
	// Message contains the email subject and body.
	Message Content `json:"message"`
}

// Message is the transport-independent email representation.
type Message struct {
	// ID is the application-owned UUIDv7 delivery tracing identifier.
	ID uuid.UUID
	// To contains primary message recipients.
	To []mail.Address
	// CC contains carbon-copy message recipients.
	CC []mail.Address
	// BCC contains blind-carbon-copy message recipients.
	BCC []mail.Address
	// Subject is the email subject.
	Subject string
	// Body is the email body.
	Body string
	// BodyFormat selects text/plain or text/html MIME rendering.
	BodyFormat BodyFormat
}

// Sender delivers one prepared email message.
type Sender interface {
	Send(context.Context, Message) error
}

// Forwarder validates API requests and forwards them as email.
type Forwarder struct {
	// sender delivers prepared messages.
	sender Sender
	// defaultBodyFormat is used when a request does not provide an override.
	defaultBodyFormat BodyFormat
}

// NewForwarder constructs a mail forwarding service.
func NewForwarder(sender Sender, defaultBodyFormat BodyFormat) *Forwarder {
	if !ValidBodyFormat(defaultBodyFormat) {
		defaultBodyFormat = BodyFormatText
	}
	return &Forwarder{sender: sender, defaultBodyFormat: defaultBodyFormat}
}

// Forward validates an incoming request and sends exactly one email message.
func (f *Forwarder) Forward(ctx context.Context, request Request) error {
	if err := ValidateRequest(request); err != nil {
		return err
	}
	if f == nil || f.sender == nil {
		return errors.New("email sender is not configured")
	}

	return f.sender.Send(ctx, Message{
		ID:         uuid.NewV7(),
		To:         addresses(request.Recipients.To),
		CC:         addresses(request.Recipients.CC),
		BCC:        addresses(request.Recipients.BCC),
		Subject:    strings.TrimSpace(request.Message.Subject),
		Body:       request.Message.Body,
		BodyFormat: resolveBodyFormat(request.Message.BodyFormat, f.defaultBodyFormat),
	})
}

// ValidateRequest reports malformed or unsupported mail requests.
func ValidateRequest(request Request) error {
	if recipientCount(request.Recipients) == 0 {
		return invalidRequest("at least one recipient is required")
	}
	if err := validateRecipients("to", request.Recipients.To); err != nil {
		return err
	}
	if err := validateRecipients("cc", request.Recipients.CC); err != nil {
		return err
	}
	if err := validateRecipients("bcc", request.Recipients.BCC); err != nil {
		return err
	}
	if strings.TrimSpace(request.Message.Subject) == "" {
		return invalidRequest("message subject is required")
	}
	if strings.TrimSpace(request.Message.Body) == "" {
		return invalidRequest("message body is required")
	}
	if !validBodyFormatOverride(request.Message.BodyFormat) {
		return invalidRequest("message body format %q is unsupported", request.Message.BodyFormat)
	}
	return nil
}

// ValidBodyFormat reports whether format is a supported concrete email body format.
func ValidBodyFormat(format BodyFormat) bool {
	return format == BodyFormatText || format == BodyFormatHTML
}

// validBodyFormatOverride reports whether format is empty or a supported request override.
func validBodyFormatOverride(format BodyFormat) bool {
	return format == "" || ValidBodyFormat(format)
}

// resolveBodyFormat applies a per-request override over the configured default.
func resolveBodyFormat(override, fallback BodyFormat) BodyFormat {
	if override != "" {
		return override
	}
	if ValidBodyFormat(fallback) {
		return fallback
	}
	return BodyFormatText
}

// validateRecipients validates each recipient in one RFC recipient class.
func validateRecipients(kind string, recipients []Recipient) error {
	for index, recipient := range recipients {
		email := strings.TrimSpace(recipient.Email)
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email {
			return invalidRequest("%s recipient %d email is invalid", kind, index)
		}
		if strings.ContainsAny(recipient.DisplayName, "\r\n") {
			return invalidRequest("%s recipient %d display name is invalid", kind, index)
		}
	}
	return nil
}

// recipientCount returns the total number of To, Cc, and Bcc recipients.
func recipientCount(recipients Recipients) int {
	return len(recipients.To) + len(recipients.CC) + len(recipients.BCC)
}

// addresses converts API recipients into canonical mail addresses.
func addresses(recipients []Recipient) []mail.Address {
	result := make([]mail.Address, 0, len(recipients))
	for _, recipient := range recipients {
		result = append(result, mail.Address{
			Name:    strings.TrimSpace(recipient.DisplayName),
			Address: strings.TrimSpace(recipient.Email),
		})
	}
	return result
}

// invalidRequest creates a validation error without exposing transport details to the application layer.
func invalidRequest(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, fmt.Sprintf(format, args...))
}

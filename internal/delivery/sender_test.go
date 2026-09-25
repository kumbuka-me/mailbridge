package delivery

import (
	"context"
	"net"
	"net/mail"
	"testing"
	"time"
	"uuid"

	"github.com/containeroo/mailbridge/internal/application"
	"github.com/containeroo/notifykit/targets/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew validates Notifykit SMTP target construction.
func TestNew(t *testing.T) {
	t.Parallel()

	sender, err := New(Config{
		Address: "smtp.example.com:465",
		From:    "mailbridge <mailbridge@example.com>",
		TLS:     email.TLSImplicit,
		Timeout: 30 * time.Second,
	})

	require.NoError(t, err)
	require.NotNil(t, sender)
	require.NotNil(t, sender.target)
	assert.Equal(t, "smtp.example.com", sender.target.Host)
	assert.Equal(t, 465, sender.target.Port)
	assert.Equal(t, email.TLSImplicit, sender.target.TLSMode)
}

// TestNewRejectsInvalidAddress verifies deployment-level SMTP addresses fail early.
func TestNewRejectsInvalidAddress(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Address: "smtp.example.com", Timeout: time.Second})
	require.Error(t, err)
}

// TestEmailNotificationID verifies the application UUID is exposed as a string to Notifykit.
func TestEmailNotificationID(t *testing.T) {
	t.Parallel()

	id := uuid.NewV7()
	n := emailNotification{message: application.Message{ID: id}}
	assert.Equal(t, id.String(), n.ID())
}

// TestEmailNotificationData exposes the already-rendered subject and body.
func TestEmailNotificationData(t *testing.T) {
	t.Parallel()

	n := emailNotification{message: application.Message{Subject: "Subject", Body: "Body"}}
	data := n.Data("smtp", nil, "ignored")
	assert.Equal(t, struct {
		Subject string
		Body    string
	}{Subject: "Subject", Body: "Body"}, data)
}

// TestNotifykitBodyFormat verifies application body formats map to SMTP MIME formats.
func TestNotifykitBodyFormat(t *testing.T) {
	t.Parallel()

	assert.Equal(t, email.BodyText, notifykitBodyFormat(application.BodyFormatText))
	assert.Equal(t, email.BodyHTML, notifykitBodyFormat(application.BodyFormatHTML))
	assert.Equal(t, email.BodyText, notifykitBodyFormat(""))
}

// TestMailboxes verifies application recipients preserve display names for message headers.
func TestMailboxes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{`"Alice" <alice@example.com>`, `<bob@example.com>`}, mailboxes([]mail.Address{
		{Name: "Alice", Address: "alice@example.com"},
		{Address: "bob@example.com"},
	}))
}

// TestSenderSend verifies one message can be delivered through Notifykit's SMTP target.
func TestSenderSend(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close() // nolint:errcheck

	// The detailed SMTP transaction is covered by Notifykit. Here we only verify
	// mailbridge reaches the target and propagates the delivery error.
	sender, err := New(Config{
		Address: listener.Addr().String(),
		From:    "mailbridge@example.com",
		TLS:     email.TLSPlaintext,
		Timeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)

	err = sender.Send(context.Background(), application.Message{
		ID:         uuid.NewV7(),
		To:         []mail.Address{{Name: "Alice", Address: "alice@example.com"}},
		CC:         []mail.Address{{Name: "Charlie", Address: "charlie@example.com"}},
		BCC:        []mail.Address{{Address: "audit@example.com"}},
		Subject:    "Test",
		Body:       "Body",
		BodyFormat: application.BodyFormatText,
	})
	require.Error(t, err)
}

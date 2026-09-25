package application

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// senderStub records the message delivered by a forwarding test.
type senderStub struct {
	message Message
	err     error
}

// Send records the supplied message and returns the configured error.
func (s *senderStub) Send(_ context.Context, message Message) error {
	s.message = message
	return s.err
}

func TestForward(t *testing.T) {
	t.Parallel()

	t.Run("builds one message with to cc and bcc recipients", func(t *testing.T) {
		t.Parallel()

		sender := &senderStub{}
		request := Request{
			Recipients: Recipients{
				To:  []Recipient{{Email: "alice@example.com", DisplayName: "Alice"}, {Email: "bob@example.com", DisplayName: "Bob"}},
				CC:  []Recipient{{Email: "charlie@example.com", DisplayName: "Charlie"}},
				BCC: []Recipient{{Email: "audit@example.com"}},
			},
			Message: Content{Subject: "Page updated", Body: "A watched page changed."},
		}

		require.NoError(t, NewForwarder(sender, BodyFormatText).Forward(context.Background(), request))
		assert.NotEqual(t, uuid.Nil(), sender.message.ID)
		assert.Equal(t, byte(7), sender.message.ID[6]>>4)
		require.Len(t, sender.message.To, 2)
		assert.Equal(t, "alice@example.com", sender.message.To[0].Address)
		assert.Equal(t, "Alice", sender.message.To[0].Name)
		require.Len(t, sender.message.CC, 1)
		assert.Equal(t, "charlie@example.com", sender.message.CC[0].Address)
		require.Len(t, sender.message.BCC, 1)
		assert.Equal(t, "audit@example.com", sender.message.BCC[0].Address)
		assert.Equal(t, "Page updated", sender.message.Subject)
		assert.Equal(t, "A watched page changed.", sender.message.Body)
		assert.Equal(t, BodyFormatText, sender.message.BodyFormat)
	})

	t.Run("uses configured html body format by default", func(t *testing.T) {
		t.Parallel()

		sender := &senderStub{}
		request := validRequest()
		request.Message.Body = "<p>Changed</p>"

		require.NoError(t, NewForwarder(sender, BodyFormatHTML).Forward(context.Background(), request))
		assert.Equal(t, BodyFormatHTML, sender.message.BodyFormat)
		assert.Equal(t, "<p>Changed</p>", sender.message.Body)
	})

	t.Run("request html overrides configured text format", func(t *testing.T) {
		t.Parallel()

		sender := &senderStub{}
		request := validRequest()
		request.Message.BodyFormat = BodyFormatHTML

		require.NoError(t, NewForwarder(sender, BodyFormatText).Forward(context.Background(), request))
		assert.Equal(t, BodyFormatHTML, sender.message.BodyFormat)
	})

	t.Run("request text overrides configured html format", func(t *testing.T) {
		t.Parallel()

		sender := &senderStub{}
		request := validRequest()
		request.Message.BodyFormat = BodyFormatText

		require.NoError(t, NewForwarder(sender, BodyFormatHTML).Forward(context.Background(), request))
		assert.Equal(t, BodyFormatText, sender.message.BodyFormat)
	})

	t.Run("empty configured format falls back to text", func(t *testing.T) {
		t.Parallel()

		sender := &senderStub{}
		require.NoError(t, NewForwarder(sender, "").Forward(context.Background(), validRequest()))
		assert.Equal(t, BodyFormatText, sender.message.BodyFormat)
	})
}

func TestForwardPropagatesSenderFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("smtp unavailable")
	sender := &senderStub{err: failure}

	err := NewForwarder(sender, BodyFormatText).Forward(context.Background(), validRequest())
	require.ErrorIs(t, err, failure)
}

func TestValidateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{name: "missing recipients", mutate: func(request *Request) { request.Recipients = Recipients{} }},
		{name: "invalid to email", mutate: func(request *Request) { request.Recipients.To[0].Email = "not-an-email" }},
		{name: "invalid cc email", mutate: func(request *Request) {
			request.Recipients.To = nil
			request.Recipients.CC = []Recipient{{Email: "bad"}}
		}},
		{name: "invalid bcc email", mutate: func(request *Request) {
			request.Recipients.To = nil
			request.Recipients.BCC = []Recipient{{Email: "bad"}}
		}},
		{name: "invalid display name", mutate: func(request *Request) { request.Recipients.To[0].DisplayName = "Alice\nBcc: bad@example.com" }},
		{name: "missing subject", mutate: func(request *Request) { request.Message.Subject = " " }},
		{name: "missing body", mutate: func(request *Request) { request.Message.Body = " " }},
		{name: "unsupported body format", mutate: func(request *Request) { request.Message.BodyFormat = "markdown" }},
	}

	require.NoError(t, ValidateRequest(validRequest()))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := validRequest()
			test.mutate(&request)
			err := ValidateRequest(request)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidRequest)
		})
	}
}

func TestValidBodyFormat(t *testing.T) {
	t.Parallel()

	assert.True(t, ValidBodyFormat(BodyFormatText))
	assert.True(t, ValidBodyFormat(BodyFormatHTML))
	assert.False(t, ValidBodyFormat(""))
	assert.False(t, ValidBodyFormat("markdown"))
}

// validRequest returns the smallest valid mail request used by tests.
func validRequest() Request {
	return Request{
		Recipients: Recipients{To: []Recipient{{Email: "alice@example.com", DisplayName: "Alice"}}},
		Message:    Content{Subject: "Updated", Body: "Changed"},
	}
}

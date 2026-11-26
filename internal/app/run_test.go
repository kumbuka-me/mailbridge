package app

import (
	"testing"

	"github.com/containeroo/notifykit/targets/email"
	"github.com/kumbuka-me/mailbridge/internal/cli"
	"github.com/stretchr/testify/assert"
)

// TestNotifykitTLSMode verifies stable mailbridge CLI values map to Notifykit modes.
func TestNotifykitTLSMode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, email.TLSRequired, notifykitTLSMode(cli.SMTPTLSStartTLS))
	assert.Equal(t, email.TLSImplicit, notifykitTLSMode(cli.SMTPTLSTLS))
	assert.Equal(t, email.TLSPlaintext, notifykitTLSMode(cli.SMTPTLSNone))
}

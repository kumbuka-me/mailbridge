package app

import (
	"testing"

	"github.com/containeroo/mailbridge/internal/flags"
	"github.com/containeroo/notifykit/targets/email"
	"github.com/stretchr/testify/assert"
)

// TestNotifykitTLSMode verifies stable mailbridge CLI values map to Notifykit modes.
func TestNotifykitTLSMode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, email.TLSRequired, notifykitTLSMode(flags.SMTPTLSStartTLS))
	assert.Equal(t, email.TLSImplicit, notifykitTLSMode(flags.SMTPTLSTLS))
	assert.Equal(t, email.TLSPlaintext, notifykitTLSMode(flags.SMTPTLSNone))
}

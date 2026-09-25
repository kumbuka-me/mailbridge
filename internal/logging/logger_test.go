package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupInstallsDefaultLogger(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })

	var output bytes.Buffer
	logger := Setup(LogFormatJSON, false, &output)

	assert.Equal(t, logger, slog.Default())
	slog.Info("ready", "event", "test")
	assert.Contains(t, output.String(), `"event":"test"`)
}

// Package logging configures structured process logging.
package logging

import (
	"io"
	"log/slog"
)

// LogFormat defines a supported structured log format.
type LogFormat string

const (
	// LogFormatText writes slog text records.
	LogFormatText LogFormat = "text"
	// LogFormatJSON writes slog JSON records.
	LogFormatJSON LogFormat = "json"
)

// Setup configures and installs the process-wide structured logger.
func Setup(format LogFormat, debug bool, output io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{}
	if debug {
		options.Level = slog.LevelDebug
	}

	var handler slog.Handler
	if format == LogFormatText {
		handler = slog.NewTextHandler(output, options)
	} else {
		handler = slog.NewJSONHandler(output, options)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// Package app composes and runs the mailbridge process.
package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	server "github.com/containeroo/httpkit/grace"
	"github.com/containeroo/notifykit/notify"
	"github.com/containeroo/notifykit/targets/email"
	"github.com/containeroo/tinyflags"
	"github.com/kumbuka-me/mailbridge/internal/application"
	"github.com/kumbuka-me/mailbridge/internal/cli"
	"github.com/kumbuka-me/mailbridge/internal/delivery"
	"github.com/kumbuka-me/mailbridge/internal/logging"
	appmetrics "github.com/kumbuka-me/mailbridge/internal/metrics"
	mailserver "github.com/kumbuka-me/mailbridge/internal/server"
)

// Run composes and runs the mailbridge process.
func Run(
	ctx context.Context,
	args []string,
	appFS fs.FS,
	version, commit string,
	stdout, stderr io.Writer,
) error {
	flags, err := cli.Parse(args, version)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(stdout, err.Error()) // nolint:errcheck
			return nil
		}
		fmt.Fprintln(stderr, err) // nolint:errcheck
		return err
	}

	logger := logging.Setup(flags.LogFormat, flags.Debug, stdout)
	setupLogger := logger.With("component", "setup")
	setupLogger.Info(
		"starting mailbridge",
		"event", "app_starting",
		"version", version,
		"commit", commit,
	)
	if len(flags.Overrides) > 0 {
		setupLogger.Info(
			"CLI Overrides",
			"event", "cli_overrides",
			"overrides", flags.Overrides.Values(),
		)
	}

	metricsRegistry := appmetrics.NewRegistry(version, commit)

	sender, err := delivery.New(
		delivery.Config{
			Address:              flags.SMTPAddress,
			From:                 flags.SMTPFrom,
			Username:             flags.SMTPUsername,
			Password:             flags.SMTPPassword,
			TLS:                  notifykitTLSMode(flags.SMTPTLS),
			InsecureSkipVerify:   flags.SMTPInsecureSkipVerify,
			Timeout:              flags.SMTPTimeout,
			ProxyFromEnvironment: true,
			Retry: notify.RetryConfig{
				Count:      flags.SMTPRetryCount,
				Backoff:    flags.SMTPRetryBackoff,
				MaxBackoff: flags.SMTPRetryMaxBackoff,
				Jitter:     flags.SMTPRetryJitter,
				Policy:     notify.DefaultRetryPolicy,
			},
			Logger: logger.With("component", "notifykit"),
		})
	if err != nil {
		setupLogger.Error("configure delivery", "event", "delivery_configuration_failed", "error", err)
		return err
	}
	proxy, err := sender.ProxyAddress()
	if err != nil {
		setupLogger.Error("resolve smtp proxy", "event", "smtp_proxy_configuration_failed", "error", err)
		return err
	}
	if proxy != "" {
		setupLogger.Info(
			"SMTP proxy configured",
			"event", "smtp_proxy_configured",
			"proxy", proxy,
		)
	}

	forwarder := application.NewForwarder(sender, flags.BodyFormat, metricsRegistry)
	handler := mailserver.New(
		mailserver.Config{
			Version:        version,
			Commit:         commit,
			Assets:         appFS,
			BodyFormat:     string(flags.BodyFormat),
			SMTPTLS:        string(flags.SMTPTLS),
			SMTPRetryCount: flags.SMTPRetryCount,
			Forwarder:      forwarder,
			APIToken:       flags.APIToken,
			Logger:         logger.With("component", "server"),
			AccessLog:      flags.AccessLog,
			RateLimit:      flags.RateLimit,
			Metrics:        metricsRegistry,
		})

	ctx, stop := server.SignalContext(ctx)
	defer stop()

	if err := server.Run(
		ctx,
		flags.ListenAddress,
		handler,
		logger.With("component", "server"),
		server.WithMaxHeaderValueCount(100),
	); err != nil {
		setupLogger.Error("start server", "event", "server_start_failed", "error", err)
		return err
	}

	return nil
}

// notifykitTLSMode maps stable mailbridge CLI values to Notifykit SMTP modes.
func notifykitTLSMode(mode cli.SMTPTLSMode) email.TLSMode {
	switch mode {
	case cli.SMTPTLSTLS:
		return email.TLSImplicit
	case cli.SMTPTLSNone:
		return email.TLSPlaintext
	default:
		return email.TLSRequired
	}
}

// Package app composes and runs the mailbridge process.
package app

import (
	"context"
	"fmt"
	"io"

	"github.com/containeroo/httpgrace/server"
	"github.com/containeroo/mailbridge/internal/application"
	"github.com/containeroo/mailbridge/internal/delivery"
	"github.com/containeroo/mailbridge/internal/flags"
	"github.com/containeroo/mailbridge/internal/logging"
	mailserver "github.com/containeroo/mailbridge/internal/server"
	"github.com/containeroo/notifykit/notify"
	"github.com/containeroo/notifykit/targets/email"
	"github.com/containeroo/tinyflags"
)

// Run composes and runs the mailbridge process.
func Run(ctx context.Context, args []string, version, commit string, stdout, stderr io.Writer) error {
	cfg, err := flags.Parse(args, version)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(stdout, err.Error()) // nolint:errcheck
			return nil
		}
		fmt.Fprintln(stderr, err) // nolint:errcheck
		return err
	}

	logger := logging.Setup(cfg.LogFormat, cfg.Debug, stdout)
	setupLogger := logger.With("component", "setup")
	setupLogger.Info(
		"starting mailbridge",
		"event", "app_starting",
		"version", version,
		"commit", commit,
	)
	if len(cfg.Overrides) > 0 {
		setupLogger.Info(
			"CLI Overrides",
			"event", "cli_overrides",
			"overrides", cfg.Overrides,
		)
	}

	sender, err := delivery.New(
		delivery.Config{
			Address:              cfg.SMTPAddress,
			From:                 cfg.SMTPFrom,
			Username:             cfg.SMTPUsername,
			Password:             cfg.SMTPPassword,
			TLS:                  notifykitTLSMode(cfg.SMTPTLS),
			InsecureSkipVerify:   cfg.SMTPInsecureSkipVerify,
			Timeout:              cfg.SMTPTimeout,
			ProxyFromEnvironment: true,
			Retry: notify.RetryConfig{
				Count:      cfg.SMTPRetryCount,
				Backoff:    cfg.SMTPRetryBackoff,
				MaxBackoff: cfg.SMTPRetryMaxBackoff,
				Jitter:     cfg.SMTPRetryJitter,
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

	forwarder := application.NewForwarder(sender, cfg.BodyFormat)
	handler := mailserver.New(mailserver.Config{
		Version:   version,
		Forwarder: forwarder,
		APIToken:  cfg.APIToken,
		Logger:    logger.With("component", "server"),
		AccessLog: cfg.AccessLog,
		RateLimit: cfg.RateLimit,
	})

	ctx, stop := server.SignalContext(ctx)
	defer stop()

	if err := server.Run(
		ctx,
		cfg.ListenAddress,
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
func notifykitTLSMode(mode flags.SMTPTLSMode) email.TLSMode {
	switch mode {
	case flags.SMTPTLSTLS:
		return email.TLSImplicit
	case flags.SMTPTLSNone:
		return email.TLSPlaintext
	default:
		return email.TLSRequired
	}
}

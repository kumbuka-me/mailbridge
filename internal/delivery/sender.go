// Package delivery adapts application mail messages to Notifykit delivery targets.
package delivery

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/mail"
	"strconv"
	"time"

	"github.com/containeroo/mailbridge/internal/application"
	"github.com/containeroo/notifykit/notify"
	"github.com/containeroo/notifykit/targets/email"
	"github.com/containeroo/notifykit/templates"
)

const receiverID notify.ReceiverID = "smtp"

// Config contains SMTP transport and retry configuration.
type Config struct {
	// Address is the SMTP server in HOST:PORT form.
	Address string
	// From is the envelope and message sender mailbox.
	From string
	// Username optionally enables SMTP authentication.
	Username string
	// Password is paired with Username.
	Password string
	// TLS selects Notifykit's SMTP TLS mode.
	TLS email.TLSMode
	// InsecureSkipVerify disables SMTP certificate verification.
	InsecureSkipVerify bool
	// Timeout bounds each complete SMTP delivery attempt.
	Timeout time.Duration
	// ProxyFromEnvironment enables HTTP_PROXY, HTTPS_PROXY, and NO_PROXY.
	ProxyFromEnvironment bool
	// Retry contains Notifykit's transient-failure retry policy.
	Retry notify.RetryConfig
	// Logger receives Notifykit's standard delivery and retry logs.
	Logger *slog.Logger
}

// Sender delivers application messages through Notifykit's SMTP target.
type Sender struct {
	// target contains immutable deployment-level SMTP configuration.
	target *email.Target
	// retry controls transient-failure retries.
	retry notify.RetryConfig
	// logger receives Notifykit's standard delivery logs.
	logger *slog.Logger
}

// New constructs an SMTP sender backed by Notifykit's email target.
func New(config Config) (*Sender, error) {
	host, portText, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, errors.New("smtp address must use HOST:PORT form")
	}
	if host == "" {
		return nil, errors.New("smtp host is required")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return nil, errors.New("smtp port is invalid")
	}

	subjectTemplate, err := templates.ParseStringTemplate("mailbridge-subject", `{{ .Subject }}`)
	if err != nil {
		return nil, err
	}
	bodyTemplate, err := templates.ParseTemplate("mailbridge-body", `{{ .Body }}`)
	if err != nil {
		return nil, err
	}

	options := []email.Option{
		email.WithHost(host),
		email.WithPort(port),
		email.WithFrom(config.From),
		email.WithTLSMode(config.TLS),
		email.WithTimeout(config.Timeout),
		email.WithDialTimeout(config.Timeout),
		email.WithSubjectTemplate(subjectTemplate),
		email.WithTemplate(bodyTemplate),
		email.WithHeader("Auto-Submitted", "auto-generated"),
	}
	if config.Username != "" || config.Password != "" {
		options = append(options, email.WithCredentials(config.Username, config.Password))
	}
	if config.InsecureSkipVerify {
		options = append(options, email.WithSkipTLSVerify())
	}
	if config.ProxyFromEnvironment {
		options = append(options, email.WithProxyFromEnvironment())
	}

	return &Sender{
		target: email.New(options...),
		retry:  config.Retry,
		logger: config.Logger,
	}, nil
}

// ProxyAddress returns the proxy authority selected from the environment, if any.
func (s *Sender) ProxyAddress() (string, error) {
	if s == nil || s.target == nil {
		return "", nil
	}
	return s.target.ProxyAddress()
}

// Send delivers one application message synchronously.
func (s *Sender) Send(ctx context.Context, message application.Message) error {
	if s == nil || s.target == nil {
		return errors.New("email target is not configured")
	}

	target := email.NewFromTarget(
		*s.target,
		email.WithTo(mailboxes(message.To)...),
		email.WithCC(mailboxes(message.CC)...),
		email.WithBCC(mailboxes(message.BCC)...),
		email.WithBodyFormat(notifykitBodyFormat(message.BodyFormat)),
	)
	notification := emailNotification{message: message}
	receiver := notify.NewReceiver(receiverID, target).WithRetry(s.retry)

	return notify.Send(ctx, notification, notify.NewReceivers(receiver), s.logger)
}

// emailNotification adapts an application message to Notifykit's notification contract.
type emailNotification struct {
	// message is the application-owned email payload.
	message application.Message
}

// ID returns the application-owned notification identifier expected by Notifykit.
func (n emailNotification) ID() string {
	return n.message.ID.String()
}

// Data exposes the already-rendered application message to Notifykit's templates.
func (n emailNotification) Data(_ string, _ map[string]any, _ string) any {
	return struct {
		Subject string
		Body    string
	}{
		Subject: n.message.Subject,
		Body:    n.message.Body,
	}
}

// notifykitBodyFormat maps the transport-independent application format to Notifykit.
func notifykitBodyFormat(format application.BodyFormat) email.BodyFormat {
	if format == application.BodyFormatHTML {
		return email.BodyHTML
	}
	return email.BodyText
}

// mailboxes renders canonical RFC mailbox strings for Notifykit.
func mailboxes(addresses []mail.Address) []string {
	result := make([]string, 0, len(addresses))
	for _, address := range addresses {
		result = append(result, address.String())
	}
	return result
}

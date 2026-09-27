// Package mailer sends transactional email over SMTP.
//
// It is deliberately small: services that need password-reset links, deadline
// reminders or receipt notices want one call, not a mail framework. Templating,
// queueing and retry belong to the caller — an outbox row plus a worker is the
// usual shape, and this package is what that worker calls at the very end.
//
// With no SMTP URL configured, New returns a logging mailer instead of failing.
// That keeps local development and CI running without a mail server, and keeps a
// missing mail server from blocking business flows in environments where email is
// a nice-to-have. Environments where delivery is mandatory should check
// Configured() at startup and refuse to boot.
package mailer

import (
	"fmt"
	"net/smtp"
	"net/url"
	"strings"

	"github.com/haipham22/govern/log"
)

// Mailer sends one message. Implementations must be safe for concurrent use.
type Mailer interface {
	Send(to, subject, body string) error
	// Configured reports whether messages actually leave the process.
	// False means Send only logs — check this at startup if delivery is mandatory.
	Configured() bool
}

// Config describes where and as whom to send.
type Config struct {
	// URL is an SMTP endpoint, e.g. smtp://localhost:1025 or
	// smtp://user:pass@mail.example.com:587. Empty disables delivery: New
	// returns a mailer that logs instead of sending.
	URL string `yaml:"smtp_url" mapstructure:"smtp_url" json:"-"`

	// From is the envelope sender and the From header.
	From string `yaml:"smtp_from" mapstructure:"smtp_from" default:"no-reply@localhost"`

	// FromName, when set, renders the header as `FromName <From>`.
	FromName string `yaml:"smtp_from_name" mapstructure:"smtp_from_name"`
}

// New builds a Mailer from cfg. It never returns an error: a missing or
// unparseable URL degrades to the logging mailer and says so in the log, because
// a malformed env var should not take a service down at startup.
func New(cfg Config) Mailer {
	if cfg.From == "" {
		cfg.From = "no-reply@localhost"
	}
	if cfg.URL == "" {
		return logMailer{from: cfg.From}
	}

	u, err := url.Parse(cfg.URL)
	if err != nil || u.Host == "" {
		log.Default().Warnw("mailer: invalid SMTP URL, falling back to log-only", "url", cfg.URL, "err", err)
		return logMailer{from: cfg.From}
	}

	m := smtpMailer{addr: u.Host, from: cfg.From, fromName: cfg.FromName}
	if u.User != nil {
		pass, _ := u.User.Password()
		m.auth = smtp.PlainAuth("", u.User.Username(), pass, u.Hostname())
	}
	return m
}

type smtpMailer struct {
	addr     string // host:port
	from     string
	fromName string
	auth     smtp.Auth // nil for servers that take no auth (Mailpit, MailHog)
}

func (m smtpMailer) Configured() bool { return true }

func (m smtpMailer) Send(to, subject, body string) error {
	if err := smtp.SendMail(m.addr, m.auth, m.from, []string{to}, m.message(to, subject, body)); err != nil {
		return fmt.Errorf("mailer: send to %s: %w", to, err)
	}
	return nil
}

func (m smtpMailer) message(to, subject, body string) []byte {
	from := m.from
	if m.fromName != "" {
		from = fmt.Sprintf("%s <%s>", m.fromName, m.from)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

type logMailer struct{ from string }

func (logMailer) Configured() bool { return false }

func (m logMailer) Send(to, subject, body string) error {
	log.Default().Infow("mailer: log-only, no SMTP URL configured",
		"from", m.from, "to", to, "subject", subject, "body", body)
	return nil
}

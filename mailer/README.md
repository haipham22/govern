# mailer

Transactional email over SMTP. One call, no mail framework.

Templating, queueing and retry stay with the caller — the usual shape is an outbox row plus a
worker, and this package is what that worker calls at the very end.

## Features

- `New(Config)` never returns an error. A missing or unparseable `URL` degrades to a **log-only**
  mailer and says so in the log, so a malformed env var cannot take a service down at startup.
- `Configured()` reports whether messages actually leave the process. Environments where delivery
  is mandatory should check it at boot and refuse to start.
- Credentials come from the URL: `smtp://user:pass@host:587` enables `PLAIN` auth,
  `smtp://localhost:1025` sends without auth (Mailpit, MailHog).
- UTF-8 plain-text body, `From`/`To`/`Subject`/`MIME-Version`/`Content-Type` headers written for you.
- Config fields carry `yaml` tags, so any config loader (viper, env, ...) populates them
  straight from the environment.

## Usage

```go
var cfg mailer.Config
// populate from your config source (yaml/env); see the table below

m := mailer.New(cfg)
if !m.Configured() {
    log.Default().Warn("email delivery disabled — log-only")
}

if err := m.Send("driver@example.com", "Schedule assigned", body); err != nil {
    // Queue a retry; do not fail the business operation on a mail error.
}
```

## Config

| Field | Suggested env var | Default | Notes |
| --- | --- | --- | --- |
| `URL` | `SMTP_URL` | — | Empty ⇒ log-only. `smtp://[user:pass@]host:port` |
| `From` | `SMTP_FROM` | `no-reply@localhost` | Envelope sender and `From` header |
| `FromName` | `SMTP_FROM_NAME` | — | Set ⇒ header renders as `FromName <From>` |

## Not in scope

Templating, HTML bodies, attachments, batching, retry/backoff, bounce handling. Add them in the
calling service, or propose a separate package — this one stays a thin SMTP write.

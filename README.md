# mailbridge

mailbridge is a small HTTP-to-SMTP bridge. It accepts a JSON request describing one email message and forwards that message through a configured SMTP server.

The project is intentionally a single Go binary with no database. SMTP delivery is synchronous: `POST /mail` returns success only after the configured SMTP server has accepted the message or the configured retry policy is exhausted.

## Request model

One API request always represents exactly one RFC email message:

```text
mailbridge request
      │
      ├── recipients
      │    ├── to
      │    ├── cc
      │    └── bcc
      │
      └── message
           ├── subject
           ├── body
           └── body_format
                │
                ▼
          one RFC email
                │
                ▼
              SMTP
```

If an application wants to send separate emails to multiple people, it should call `POST /mail` once for each message. Multiple recipients inside one request are recipients of the same email message.

## API

### `POST /mail`

The request must use bearer authentication:

```text
Authorization: Bearer <api-token>
```

Example:

```json
{
  "recipients": {
    "to": [
      {
        "email": "alice@example.com",
        "display_name": "Alice"
      },
      {
        "email": "bob@example.com",
        "display_name": "Bob"
      }
    ],
    "cc": [
      {
        "email": "charlie@example.com",
        "display_name": "Charlie"
      }
    ],
    "bcc": []
  },
  "message": {
    "subject": "Page updated: Runbook",
    "body": "<p>A watched page changed.</p>",
    "body_format": "html"
  }
}
```

`recipients.to`, `recipients.cc`, and `recipients.bcc` are arrays. At least one recipient across the three arrays is required. `display_name` is optional.

`message.subject` and `message.body` are required. `message.body_format` is optional and accepts `text` or `html`. When omitted, mailbridge uses the global `--body-format` setting. A request-level value always overrides the global default.

The message body is sent as supplied. mailbridge does not sanitize, escape, wrap, or otherwise rewrite HTML.

Successful SMTP acceptance returns `204 No Content`. Invalid requests return `400`, unauthorized requests return `401`, rate-limited requests return `429`, and exhausted SMTP delivery failures return `502`.

When rate limiting is enabled, the limit is shared across all authenticated `POST /mail` requests. Requests exceeding the configured rate return `429 Too Many Requests` with `Retry-After: 1`.

### `GET /healthz`

The health endpoint is unauthenticated and returns `200 OK`:

```json
{ "status": "ok" }
```

### `GET /metrics`

The Prometheus endpoint is unauthenticated and returns metrics in the Prometheus text exposition format. mailbridge uses a private registry, so only its explicitly registered application, Go runtime, and process collectors are exposed.

## Metrics

mailbridge exposes these application metrics in addition to the standard Go runtime and process collectors:

| Metric                                       | Type      | Description                                            |
| -------------------------------------------- | --------- | ------------------------------------------------------ |
| `mailbridge_build_info`                      | gauge     | Build identity labeled by `version` and `commit`.      |
| `mailbridge_mail_requests_total`             | counter   | Completed `POST /mail` requests by response `code`.    |
| `mailbridge_mail_request_duration_seconds`   | histogram | Complete `POST /mail` request duration.                |
| `mailbridge_email_deliveries_total`          | counter   | Completed email delivery operations.                   |
| `mailbridge_email_delivery_errors_total`     | counter   | Email delivery operations that finished with an error. |
| `mailbridge_email_delivery_duration_seconds` | histogram | Complete email delivery operation duration.            |

Only `POST /mail` is instrumented at the HTTP layer. Infrastructure endpoints such as `/metrics`, `/healthz`, `/readyz`, and `/version` do not contribute to mail request metrics. Delivery metrics are recorded only after request validation succeeds and the SMTP delivery operation is attempted.

## Email semantics

A single request produces one email message.

For this request:

```text
To:  Alice, Bob
Cc:  Charlie
Bcc: Audit
```

mailbridge sends one SMTP message with Alice and Bob in the `To` header, Charlie in the `Cc` header, and Audit only in the SMTP envelope. Bcc recipients are never written to a `Bcc` message header.

If Alice and Bob should receive independent emails, send two requests instead.

## Configuration

Configuration uses `tinyflags`. Environment variables use the `MAILBRIDGE__` prefix.

| Flag                          | Environment variable                    | Default          | Description                                                   |
| ----------------------------- | --------------------------------------- | ---------------- | ------------------------------------------------------------- |
| `--listen-address`            | `MAILBRIDGE__LISTEN_ADDRESS`            | `127.0.0.1:8080` | HTTP listen address.                                          |
| `--api-token`                 | `MAILBRIDGE__API_TOKEN`                 | required         | Bearer token required by `POST /mail`.                        |
| `--rate-limit`                | `MAILBRIDGE__RATE_LIMIT`                | `0`              | Maximum mail requests per second; `0` disables rate limiting. |
| `--smtp-address`              | `MAILBRIDGE__SMTP_ADDRESS`              | required         | SMTP server as `HOST:PORT`.                                   |
| `--smtp-from`                 | `MAILBRIDGE__SMTP_FROM`                 | required         | Sender mailbox; a display name is allowed.                    |
| `--smtp-username`             | `MAILBRIDGE__SMTP_USERNAME`             | empty            | SMTP username.                                                |
| `--smtp-password`             | `MAILBRIDGE__SMTP_PASSWORD`             | empty            | SMTP password.                                                |
| `--smtp-tls`                  | `MAILBRIDGE__SMTP_TLS`                  | `starttls`       | `starttls`, `tls`, or `none`.                                 |
| `--smtp-insecure-skip-verify` | `MAILBRIDGE__SMTP_INSECURE_SKIP_VERIFY` | `false`          | Disable SMTP TLS certificate verification.                    |
| `--smtp-timeout`              | `MAILBRIDGE__SMTP_TIMEOUT`              | `30s`            | Timeout for each SMTP delivery attempt.                       |
| `--smtp-retry-count`          | `MAILBRIDGE__SMTP_RETRY_COUNT`          | `3`              | Retries after the initial SMTP attempt.                       |
| `--smtp-retry-backoff`        | `MAILBRIDGE__SMTP_RETRY_BACKOFF`        | `1s`             | Delay before the first SMTP retry.                            |
| `--smtp-retry-max-backoff`    | `MAILBRIDGE__SMTP_RETRY_MAX_BACKOFF`    | `30s`            | Maximum local exponential retry backoff.                      |
| `--smtp-retry-jitter`         | `MAILBRIDGE__SMTP_RETRY_JITTER`         | `true`           | Apply full jitter to retry delays.                            |
| `--body-format`               | `MAILBRIDGE__BODY_FORMAT`               | `text`           | Default email body format: `text` or `html`.                  |
| `--log-format`                | `MAILBRIDGE__LOG_FORMAT`                | `json`           | `json` or `text`.                                             |
| `--debug`                     | `MAILBRIDGE__DEBUG`                     | `false`          | Enable debug logging, including retry waits.                  |
| `--access-log`                | `MAILBRIDGE__ACCESS_LOG`                | `false`          | Log HTTP requests.                                            |

SMTP authentication is disabled when both username and password are empty. If one is supplied, both are required.

### Body format

The global default is plain text:

```sh
--body-format text
```

To make HTML the deployment-wide default:

```sh
--body-format html
```

or set `MAILBRIDGE__BODY_FORMAT=html`. The resulting MIME type is `text/plain; charset=utf-8` for `text` and `text/html; charset=utf-8` for `html`.

### SMTP proxy

SMTP connections honor the standard proxy environment variables `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY`, including lowercase variants. Notifykit opens an HTTP `CONNECT` tunnel through the selected proxy and performs SMTP/TLS inside that tunnel.

For implicit TLS (`--smtp-tls=tls`), `HTTPS_PROXY` is preferred and `HTTP_PROXY` is used as a fallback. For STARTTLS/plain SMTP, `HTTP_PROXY` is preferred and `HTTPS_PROXY` is used as a fallback. `NO_PROXY` bypasses proxying for matching SMTP hosts.

Example:

```sh
export HTTPS_PROXY='http://proxy.example.com:3128'
export NO_PROXY='localhost,127.0.0.1,.internal.example.com'
```

Proxy credentials may be supplied in the proxy URL. Logs only show the proxy authority and never log proxy credentials.

## Retry behavior

mailbridge uses `github.com/containeroo/notifykit` for SMTP delivery, TLS, MIME rendering, proxy tunneling, retry classification, retries, and standard delivery logging.

The default retry policy retries transport/network failures, timeouts, and temporary SMTP `4xx` replies. SMTP `5xx` replies, certificate verification failures, invalid configuration, and other permanent failures are not retried.

With the defaults, one request may make up to four SMTP attempts: one initial attempt plus three retries. Backoff starts at `1s`, doubles up to `30s`, and uses full jitter.

Retries are synchronous. One slow request does not block other HTTP requests; Go serves each request concurrently.

## Logging

A successful request produces Notifykit's target-delivery record plus an application-level success record:

```text
msg="notification target delivered" component=notifykit receiver=smtp targetType=email attempts=1 status=sent
msg="email accepted by SMTP server" component=server event=email_delivery_succeeded to_count=2 cc_count=1 bcc_count=0
```

The success message intentionally says **accepted by SMTP server** rather than delivered: SMTP acceptance does not prove the message reached every recipient's inbox.

Recipient email addresses and SMTP/proxy passwords are not included in normal delivery logs. With `--debug`, Notifykit also logs individual attempts and retry waits.

## Run

```sh
mailbridge \
  --listen-address 0.0.0.0:8080 \
  --api-token 'change-me' \
  --smtp-address smtp.example.com:587 \
  --smtp-from 'mailbridge <mailbridge@example.com>' \
  --smtp-username mailbridge \
  --smtp-password 'secret' \
  --smtp-tls starttls
```

## Send a test email

```sh
export MAILBRIDGE__LISTEN_ADDRESS=127.0.0.1:8080
export MAILBRIDGE__API_TOKEN='change-me'
export TEST_EMAIL='alice@example.com'
```

```sh
curl --fail-with-body \
  --request POST \
  --url "http://${MAILBRIDGE__LISTEN_ADDRESS}/mail" \
  --header "Authorization: Bearer ${MAILBRIDGE__API_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data @- <<JSON
{
  "recipients": {
    "to": [
      {
        "email": "${TEST_EMAIL}",
        "display_name": "Test Recipient"
      }
    ],
    "cc": [],
    "bcc": []
  },
  "message": {
    "subject": "mailbridge test",
    "body": "<h1>mailbridge test</h1><p>This is a test HTML email.</p>",
    "body_format": "html"
  }
}
JSON
```

A successful SMTP delivery returns `204 No Content`.

## Development

```sh
make fmt
make vet
make test
make test-race
make lint
make build
```

The entry point only calls `app.Run`. `internal/app` owns process composition, `internal/application` owns request validation and conversion into one email message, `internal/delivery` adapts application messages to Notifykit, `internal/handler` owns endpoint behavior, `internal/metrics` owns the private Prometheus registry and observations, `internal/middleware` owns mail request instrumentation and other cross-cutting HTTP concerns, and `internal/server` owns route construction. Notifykit owns SMTP, TLS, MIME rendering, proxy tunneling, retry classification, and delivery logging.

## Message identity

Each accepted request gets one UUIDv7 message identifier generated by mailbridge. Notifykit treats that identifier as an application-owned string and preserves it across the initial attempt, retries, and final delivery log.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).

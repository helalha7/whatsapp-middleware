# WhatsApp Middleware

A Go service that receives WhatsApp Cloud API webhooks, normalizes inbound messages, prevents duplicate webhook handoff, and processes messages asynchronously through a worker pool.

The current MVP behavior echoes inbound text messages back to the sender. The messaging processor is intended to coordinate AI-generated responses later.

## Message Flow

```text
Meta webhook
    ↓
Signature verification and normalization
    ↓
Receipt idempotency check
    ↓
Buffered in-memory queue
    ↓
Worker pool
    ↓
Messaging processor
    ├── Tenant and access-token lookup
    ├── Conversation upsert
    ├── Active chat-session selection
    ├── Inbound message persistence and session activity update
    ├── Active-session history lookup
    ├── WhatsApp sender
    └── Outbound message persistence
            ↓
      Echoed text response
```

## Package Responsibilities

| Package | Responsibility |
| --- | --- |
| `cmd` | Configuration, dependency wiring, server and worker lifecycle |
| `internal/listener` | Meta verification, signature validation, payload parsing and normalization |
| `internal/event` | Normalized inbound event types |
| `internal/data` | Shared persistence errors, MySQL inspection and list pagination primitives |
| `internal/idempotency` | Deduplicates inbound webhook handoff; it does not track processing completion |
| `internal/conversation` | Persists tenant-scoped customer conversations |
| `internal/chatsession` | Starts or continues inactivity-based chatbot sessions |
| `internal/message` | Persists inbound and outbound messages and updates conversation activity |
| `internal/queue` | Generic buffered in-memory queue |
| `internal/worker` | Generic concurrent worker pool |
| `internal/messaging` | Coordinates inbound message processing and responses |
| `internal/sender` | Sends messages through the WhatsApp Cloud API |
| `internal/tenant` | Tenant model, persistence and HTTP endpoints |
| `internal/httpx` | Shared HTTP, JSON and query-parameter helpers |

## Requirements

- Go version declared in `go.mod`
- MySQL
- [Goose](https://github.com/pressly/goose) for migrations
- A Meta app with WhatsApp Cloud API credentials

Install Goose if needed:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

## Configuration

Copy the example environment file:

```bash
cp .env.example .env
```

Configure these values:

| Variable | Description |
| --- | --- |
| `verify_token` | Token chosen for Meta webhook verification |
| `app_secret` | Meta application secret used to verify webhook signatures |
| `db_dsn` | MySQL DSN used by the service |
| `ai_session_idle_timeout` | Inactivity before a new chat session starts; defaults to `10m` |

Do not commit `.env` or real access tokens.

## Database Setup

Create the database, then run all migrations using the same DSN configured for the application:

```bash
goose -dir migrations mysql "user:password@tcp(localhost:3306)/whatsapp_middleware?parseTime=true" up
```

Check migration status:

```bash
goose -dir migrations mysql "user:password@tcp(localhost:3306)/whatsapp_middleware?parseTime=true" status
```

## Run the Service

Development mode loads `.env` automatically:

```bash
go run ./cmd
```

The server listens on port `8080`.

For production mode, provide environment variables externally:

```bash
go run ./cmd -env=production
```

## Create a Tenant

Register the WhatsApp business account before sending webhook messages:

```bash
curl --request POST http://localhost:8080/v1/tenants \
  --header 'Content-Type: application/json' \
  --data '{
    "name": "Demo Business",
    "waba_id": "YOUR_WABA_ID",
    "phone_number_id": "YOUR_PHONE_NUMBER_ID",
    "display_phone_number": "YOUR_DISPLAY_PHONE_NUMBER",
    "access_token": "YOUR_ACCESS_TOKEN"
  }'
```

The `waba_id` must match `entry[].id` in incoming Meta webhook payloads.

> Tenant administration endpoints are currently unauthenticated and must not be exposed publicly in production.

## Chat Sessions

Inbound and outbound messages are assigned to the same chat session. Both
directions update session activity, and a new session begins after the
configured inactivity timeout. Session history queries return both directions
for future AI context. The temporary echo response renders only inbound text to
avoid recursively embedding earlier transcript echoes.

## Configure the Meta Webhook

Configure the callback URL as:

```text
https://YOUR_PUBLIC_HOST/v1/webhooks
```

Use the same value configured as `verify_token`, subscribe to the `messages` field, and ensure Meta can reach the service over HTTPS.

Webhook notifications must include a valid `X-Hub-Signature-256` generated with the configured Meta app secret.
Malformed notifications for supported message types are rejected, while unrelated changes and unsupported message types are deliberately acknowledged and ignored.

## Test and Validate

```bash
go fmt ./...
go test ./...
go vet ./...
goose -dir migrations validate
```

## Current Limitations

- The queue is in memory and messages are lost if the process exits before processing them.
- Processing failures are logged but not retried.
- Only text messages currently receive a response; the temporary response echoes all customer messages in the active chat session.
- Interactive messages are normalized but currently ignored by the messaging processor.
- Processing order is not guaranteed per conversation.
- Tenant access tokens are stored in MySQL as plaintext.

See [TECHNICAL_ROADMAP.md](TECHNICAL_ROADMAP.md) for prioritized production work and completion criteria.

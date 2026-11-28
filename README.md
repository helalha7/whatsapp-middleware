# WhatsApp Middleware

A Go service that receives WhatsApp Cloud API webhooks, normalizes inbound messages, prevents duplicate webhook handoff, and processes messages asynchronously through a worker pool.

The current MVP behavior echoes inbound text messages back to the sender. The planned business information, service inquiry, and WhatsApp location replies are described below.

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

## Planned Business Information and Service Replies

### Tenant business data

The middleware stores one business information record per WhatsApp tenant and supports multiple services per tenant. The messaging workflow uses the tenant resolved from the inbound webhook to load the profile, opening hours, and active services.

| Workflow tool | Data returned |
| --- | --- |
| `get_business_information` | Business name, description, street address, timezone, weekly opening hours, walk-in policy, and optional map coordinates |
| `get_services` | Active services with their ID, name, optional description, duration in minutes, price in ILS, and effective weekly hours |

### Business database schema

Create these tables in the middleware MySQL database alongside `tenants`. The examples assume `tenants.id` is a signed `BIGINT`. Match the actual type and signedness in the migration.

```sql
CREATE TABLE business_information (
    tenant_id BIGINT PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    description TEXT NULL,
    address VARCHAR(255) NULL,
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Jerusalem',
    walk_in_policy VARCHAR(500) NULL,
    latitude DECIMAL(8,6) NULL,
    longitude DECIMAL(9,6) NULL,

    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CHECK (
        (latitude IS NULL AND longitude IS NULL)
        OR (
            latitude IS NOT NULL AND longitude IS NOT NULL
            AND latitude BETWEEN -90 AND 90
            AND longitude BETWEEN -180 AND 180
        )
    )
);

CREATE TABLE business_hours (
    tenant_id BIGINT NOT NULL,
    day_of_week TINYINT UNSIGNED NOT NULL,
    opens_at TIME NOT NULL,
    closes_at TIME NOT NULL,

    PRIMARY KEY (tenant_id, day_of_week, opens_at),
    FOREIGN KEY (tenant_id) REFERENCES business_information(tenant_id),
    CHECK (day_of_week BETWEEN 0 AND 6),
    CHECK (opens_at < closes_at)
);

CREATE TABLE services (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(120) NOT NULL,
    description VARCHAR(500) NULL,
    duration_minutes SMALLINT UNSIGNED NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CHECK (duration_minutes > 0),
    CHECK (price >= 0)
);

CREATE TABLE service_hours (
    service_id BIGINT NOT NULL,
    day_of_week TINYINT UNSIGNED NOT NULL,
    opens_at TIME NOT NULL,
    closes_at TIME NOT NULL,

    PRIMARY KEY (service_id, day_of_week, opens_at),
    FOREIGN KEY (service_id) REFERENCES services(id),
    CHECK (day_of_week BETWEEN 0 AND 6),
    CHECK (opens_at < closes_at)
);
```

`business_information.tenant_id` is both the primary key and a foreign key, so each tenant can have at most one profile. Create the profile in the same transaction as tenant registration, using the registered name, and backfill a profile for existing tenants. The address can be filled in later. Each service has its own ID and shares its tenant's `tenant_id` with that tenant's other services.

Use `0` for Sunday through `6` for Saturday in both hours tables, matching Go's `time.Weekday`. Store local times in the business profile's timezone. Multiple rows for one day represent separate opening intervals.

Store prices directly in shekels. For example, a haircut that costs ₪60.00 has `price = 60.00`. Scope service reads and updates by `tenant_id`. Return active services to the workflow and set `is_active` to false when retiring a service so existing bookings can still reference its ID.

### Service opening hours

- A business is closed on days with no `business_hours` rows.
- A service with no `service_hours` rows follows the business's weekly hours.
- Once a service has its own hours, those rows define its full weekly schedule. A day omitted from that schedule is unavailable for that service.
- The effective hours are the overlap between the service's schedule and the business's hours. For example, if the shop opens Friday from 09:00 to 14:00 and beard trims run Friday from 10:00 to 12:00, beard trims are available from 10:00 to 12:00.
- Scope service hours reads and updates through `services.tenant_id`. For appointment booking, the full service duration must fit within an effective opening interval.

### Configuring the business location

The business settings form has a street address and an optional `map_pin` field. The owner can paste a `latitude, longitude` pair or click **Use my current location** while at the shop entrance. The browser requests permission and fills the same field using `navigator.geolocation.getCurrentPosition()`. The owner reviews the value and submits the form explicitly. Manual entry is available when location permission is denied or setup happens remotely.

On save, the middleware validates that both coordinates are present and finite when a pin is provided. Latitude must be between −90 and 90, and longitude between −180 and 180. Store them separately in `business_information.latitude DECIMAL(8,6)` and `business_information.longitude DECIMAL(9,6)`. Clearing the pin sets both columns to `NULL`. The street address remains a separate field and provides the text fallback.

### Replying through WhatsApp

- For a service inquiry, the application workflow uses `get_services` to fetch the tenant's active service catalog and sends a text reply. For questions about when a service is offered, use its effective weekly hours.
- For general business questions such as opening hours, the workflow uses `get_business_information` and sends a text reply.
- When the customer asks where the business is, `internal/messaging` requests a location reply if both coordinates are available. The reply uses the business name, address, and coordinates from the tenant's business profile. When the pin is missing, send the street address as text if available. If neither is configured, tell the customer the location is unavailable.
- `internal/sender` exposes `SendLocationMessage` alongside `SendTextMessage`. It serializes the native WhatsApp `location` payload and sends it through the existing WhatsApp messages endpoint. Record the outbound reply in the active chat session with its message type.

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
GOOSE_DRIVER=mysql \
GOOSE_DBSTRING="user:password@tcp(localhost:3306)/whatsapp_middleware?parseTime=true" \
GOOSE_MIGRATION_DIR=migrations \
goose up
```

Check migration status:

```bash
GOOSE_DRIVER=mysql \
GOOSE_DBSTRING="user:password@tcp(localhost:3306)/whatsapp_middleware?parseTime=true" \
GOOSE_MIGRATION_DIR=migrations \
goose status
```

Migration filenames use Goose timestamp versions so developers can create
migrations concurrently without colliding on the next sequence number. Create
new migrations without the sequential `-s` option:

```bash
goose -dir migrations create add_example_table sql
```

Never rename or modify a migration after it has been applied to a shared
environment.

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
- Only text messages currently receive a response. The temporary response echoes all customer messages in the active chat session.
- Interactive messages are normalized but currently ignored by the messaging processor.
- Processing order is not guaranteed per conversation.
- Tenant access tokens are stored in MySQL as plaintext.

See [TECHNICAL_ROADMAP.md](TECHNICAL_ROADMAP.md) for prioritized production work and completion criteria.

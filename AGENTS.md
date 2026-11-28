# Repository Guide for Coding Agents

## Purpose

This service receives WhatsApp Cloud API webhooks and routes normalized inbound messages through an in-memory queue and worker pool. The current messaging workflow echoes text; AI response generation will be introduced behind the messaging layer later.

The next planned workflow adds tenant business profiles, weekly business and service hours, service catalogs, and native WhatsApp location replies. The schema and behavior documented in `README.md` are the source of truth for that work; do not describe planned functionality as already implemented.

Read `README.md` for setup and system behavior. Read `TECHNICAL_ROADMAP.md` before making architectural or reliability changes.

## Architecture Boundaries

- `cmd`: composition root only; create dependencies and manage lifecycle here.
- `internal/listener`: verify and normalize inbound Meta webhooks. Do not place response-generation logic here.
- `internal/event`: transport-neutral normalized event types.
- `internal/idempotency`: persistence for received message IDs.
- `internal/queue`: generic queue implementation; do not add WhatsApp-specific logic.
- `internal/worker`: generic concurrency; do not import tenant, sender, listener or Meta-specific packages.
- `internal/messaging`: coordinate message workflows, including future AI calls.
- `internal/sender`: own outbound WhatsApp HTTP requests and Meta response handling.
- `internal/tenant`: own tenant domain data and persistence.
- Business profile, hours, and service-catalog persistence belong in dedicated domain packages; do not put their query logic in `listener`, `sender`, or generic queue/worker packages.
- `internal/httpx`: reusable HTTP response and decoding helpers.

Keep dependencies pointing toward small interfaces defined by the package that consumes them. Wire concrete implementations in `cmd/application.go`.

## Required Checks

Run these after changing Go code:

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

Run this after changing migrations:

```bash
goose -dir migrations validate
```

## Database Rules

- Add a new numbered Goose migration; never modify a migration that may already be applied.
- Use timestamp-versioned migration filenames created without Goose's sequential `-s` option.
- Keep expected persistence errors in `internal/data/errors.go`.
- Translate known driver errors in stores and return unexpected errors unchanged.
- Preserve context-aware database calls.
- Never expose tenant access tokens through JSON or logs.

### Business and service schema

- Implement the README schema with `business_information`, `business_hours`, `services`, and `service_hours`; keep foreign-key column types and signedness identical to the referenced keys.
- `business_information.tenant_id` is both its primary key and a foreign key to `tenants.id`, enforcing one business profile per tenant.
- Create the business profile in the same transaction as tenant registration, initialize its name from the tenant name, and backfill profiles for existing tenants when introducing the schema.
- `business_hours` belongs to a business profile. `services` belongs to a tenant. `service_hours` belongs to a service.
- Scope every service and service-hours read or mutation by `tenant_id`, including lookups that begin with a service ID. Do not allow cross-tenant access.
- Use `0` for Sunday through `6` for Saturday, matching Go's `time.Weekday`. Reject invalid days and intervals where `opens_at >= closes_at`.
- Interpret hours in the IANA timezone stored on `business_information`; do not treat stored `TIME` values as UTC.
- Allow multiple opening intervals per day. No business-hours rows means the business is closed that day.
- A service with no `service_hours` rows inherits business hours. Once any service-specific hours exist, they define the service's complete weekly schedule; its effective availability is their overlap with business hours.
- Store service prices directly in ILS using `DECIMAL(10,2)`, require nonnegative prices and positive durations, and retire services with `is_active = false` rather than deleting records needed by future bookings.
- Store latitude and longitude as an all-or-nothing pair. Validate finite values and ranges of `[-90, 90]` and `[-180, 180]`; clearing the map pin sets both columns to `NULL`.
- Keep the street address independent from coordinates so it remains available as a text fallback.

## Webhook Rules

- Verify `X-Hub-Signature-256` before decoding or processing payload content.
- Keep request body limits in place.
- Preserve duplicate-delivery handling.
- Return `500` when a temporary internal failure should cause Meta to retry.
- Unsupported webhook changes should be acknowledged and ignored deliberately, not accidentally treated as successful business processing.

## Queue and Worker Rules

- The current queue is intentionally in memory for the MVP.
- Create queues in the composition root and inject publishers/consumers; do not add global channels.
- Stop HTTP publishers before closing the queue.
- Close the queue and let workers drain during graceful shutdown.
- Do not describe the current channel as durable or exactly-once delivery.

## Messaging and AI Rules

- `messaging.Processor` owns workflow coordination.
- A future `ai` package should interpret content and generate responses only.
- AI code must not call WhatsApp directly; delivery remains in `sender`.
- Business-information and service tools return tenant-scoped domain data to the messaging workflow; they must not send WhatsApp messages themselves.
- General business questions use the business profile and weekly hours. Service inquiries use active services and their effective hours.
- Location requests use a native WhatsApp location reply only when both coordinates are present. Otherwise send the street address as text, or state that the location is unavailable when neither is configured.
- `internal/sender` owns serialization and delivery of both text and native WhatsApp location payloads. Persist outbound replies in the active chat session with the correct message type.
- Do not log prompts, message content or credentials without an explicit privacy decision.
- Preserve or deliberately design per-conversation ordering before adding persistent AI conversation state.

## Scope and Safety

- Keep changes focused and consistent with existing naming and formatting.
- Do not commit `.env`, access tokens, app secrets or real customer payloads.
- Do not discard unrelated working-tree changes.
- Update `README.md` and `TECHNICAL_ROADMAP.md` when behavior or architectural decisions change.

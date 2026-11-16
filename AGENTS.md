# Repository Guide for Coding Agents

## Purpose

This service receives WhatsApp Cloud API webhooks and routes normalized inbound messages through an in-memory queue and worker pool. The current messaging workflow echoes text; AI response generation will be introduced behind the messaging layer later.

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
- Keep expected persistence errors in `internal/data/errors.go`.
- Translate known driver errors in stores and return unexpected errors unchanged.
- Preserve context-aware database calls.
- Never expose tenant access tokens through JSON or logs.

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
- Do not log prompts, message content or credentials without an explicit privacy decision.
- Preserve or deliberately design per-conversation ordering before adding persistent AI conversation state.

## Scope and Safety

- Keep changes focused and consistent with existing naming and formatting.
- Do not commit `.env`, access tokens, app secrets or real customer payloads.
- Do not discard unrelated working-tree changes.
- Update `README.md` and `TECHNICAL_ROADMAP.md` when behavior or architectural decisions change.

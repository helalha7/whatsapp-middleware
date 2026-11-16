# Technical Roadmap

The current architecture is suitable for development and an MVP. This document tracks the work required before treating the service as production-ready.

## Priority 1: Message Reliability

- Replace the in-memory channel with a durable queue when moving beyond the MVP.
- Add retries with exponential backoff for temporary database, AI, and Meta API failures.
- Add dead-letter handling for messages that repeatedly fail.
- Track message processing state separately from webhook receipt state.
- Prevent the gap where a receipt is stored but the process exits before the message is queued.
- Define how to handle the ambiguous case where Meta accepts an outbound message but the HTTP response is lost.

### Completion criteria

- A process restart does not lose accepted messages.
- Failed work is retried and observable.
- Duplicate delivery does not create duplicate business effects.

## Priority 2: Conversation Ordering

- Preserve ordering for messages from the same tenant and WhatsApp user.
- Prevent concurrent workers from updating the same conversation state.
- Define a conversation key, likely `tenant_id + sender_wa_id`.

### Completion criteria

- Messages from one conversation are processed in order.
- Different conversations can still be processed concurrently.

## Priority 3: Messaging Workflow

- Decide how interactive button and list replies should be processed.
- Do not report ignored message types as successfully processed.
- Add support for required media types such as images, audio, documents, and video.
- Introduce an AI responder interface in `internal/messaging` when AI integration begins.
- Persist conversation context required by the AI workflow.

### Completion criteria

- Every accepted message type has explicit processing behavior.
- Unsupported types are logged and measured clearly.
- AI generation remains separate from WhatsApp delivery.

## Priority 4: Security

- Add authentication and authorization to tenant administration endpoints.
- Encrypt tenant access tokens or move them to a secrets manager.
- Define access-token rotation and revocation procedures.
- Avoid including credentials or message contents in operational logs.

### Completion criteria

- Public callers cannot create or inspect tenants without authorization.
- Access tokens are not stored as plaintext application data.

## Priority 5: WhatsApp Sender

- Parse successful Meta responses and retain the outbound WhatsApp message ID.
- Parse Meta error responses into a structured error type.
- Make the Graph API version configurable.
- Drain successful HTTP response bodies so connections can be reused.
- Handle Meta rate limits and retryable status codes.

### Completion criteria

- Outbound requests expose the Meta message ID and structured failures.
- Retry decisions can be made without parsing error strings.

## Priority 6: HTTP and Validation

- Make `ReadIdParam` return an error for IDs less than one.
- Reject request bodies containing multiple JSON documents.
- Make webhook error responses consistent.
- Add middleware for panic recovery, request IDs, and structured logging.
- Review webhook behavior for malformed signed events to avoid unnecessary retry loops.

## Priority 7: Database Operations

- Configure maximum open connections, idle connections, and connection lifetime.
- Add a retention policy for `received_inbound_messages`.
- Add integration tests that execute migrations and stores against MySQL.
- Review indexing as message and tenant volume grows.

## Priority 8: Tests and Observability

- Add worker-pool concurrency and shutdown tests.
- Add HTTP server lifecycle tests.
- Add an integration test covering webhook receipt through outbound Meta request.
- Add metrics for queue depth, processing duration, failures, retries, and duplicates.
- Add health and readiness endpoints for the HTTP server, database, and future queue.

## Current Temporary Decisions

- The queue is an in-memory buffered Go channel.
- Processing failures are logged but not retried.
- Text messages are echoed; interactive messages are currently ignored by the messaging processor.
- Receipt idempotency prevents repeated webhook handoff but does not guarantee successful processing.
- The service runs as a single process and does not guarantee per-conversation ordering.

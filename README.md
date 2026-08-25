# Sky Stream Aggregate

**Status: engineering beta.** A bounded single-process Go event aggregation service for small keyed numeric streams.

## Implemented

- `POST /v1/events` accepts `{ "key": string, "value": number }`.
- Keys are bounded to 64 safe characters with a 256-key cardinality limit.
- Values must be finite and within ±1e12.
- Concurrency-safe per-key count and sum aggregation.
- Total and rejected-event counters.
- Strict single-object JSON parsing with unknown-field rejection and 4 KiB request bound.
- `/healthz`, `/readyz`, and `GET /v1/stats` endpoints.
- HTTP server timeouts and graceful shutdown.
- Go 1.26 fmt/vet/test/race/govulncheck/build gates and non-root container smoke tests.

## Scope limitations

This is an **in-memory single-node aggregator**, not a distributed stream platform. It does not provide Kafka/Pulsar integration, durable offsets, event replay, windows by event time, watermarks, exactly-once processing, persistence, partitioning, cross-node coordination, tenant isolation, authentication, HA, or production deployment.

All state resets when the process restarts.

## SKYCOIN4444 integration

Use this service for bounded ephemeral counters/sums or as a reference stream-processing contract. Durable event storage, broker consumption, identity, persistence, and distributed processing must remain separately verified concerns.

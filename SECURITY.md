# Security

Sky Stream Aggregate is an engineering-beta, single-process event aggregation service.

Implemented controls include bounded request bodies, strict JSON decoding, bounded key cardinality, finite numeric-value limits, concurrency-safe state, HTTP server timeouts, graceful shutdown, race tests, vulnerability scanning, and a non-root distroless image.

This service does not implement authentication, authorization, tenant isolation, rate limiting, persistence, encryption at rest, broker authentication, durable audit logs, WAF/DDoS protection, or distributed consistency. All aggregation state is ephemeral and lost on restart.

Do not expose this beta as a public multi-tenant telemetry or financial aggregation service without adding and validating the missing controls and domain-specific numerical requirements.

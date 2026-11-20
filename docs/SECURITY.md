# Security and production boundaries

This repository is an educational portfolio implementation. It is not a certified privacy, consent, or regulatory-compliance product.

## Implemented safeguards

- Every data query includes a tenant_id predicate; consent lookups, revocation, and audit reads are tenant-scoped.
- SQL is parameterized through pgx; request bodies reject unknown JSON fields and are size-limited.
- Consent writes and corresponding audit events are committed in a single database transaction.
- Authorization checks fail closed: only a matching granted record returns allowed=true.
- The container runs as a non-root user and the HTTP server configures read/write/idle timeouts.

## Important limitations

- X-Tenant-ID is an untrusted tenant selector, not authentication. The demo has no identity provider, signed token validation, user-to-tenant membership checks, or role-based authorization. A production deployment must derive tenant and actor identity from verified credentials at a trusted gateway or authentication middleware, never trust client-supplied identity headers directly.
- The API does not connect to downstream data systems or intercept all data access. Integrations must call the authorization endpoint or use an equivalent server-side policy library.
- The audit table is append-only by application convention, not immutable against a database administrator. Production audit evidence needs restricted database roles, retention controls, monitoring, and possibly an external tamper-evident sink.
- No consent UI, policy versioning, jurisdiction rules, retention/deletion orchestration, data-subject verification, encryption key management, rate limiting, or formal compliance certification is included.
- Use synthetic identifiers and data only. Do not submit real personal information to this demo.

Before production use, add authentication/authorization, tenant provisioning, secret management, TLS, rate limits, observability, backup/recovery procedures, security review, privacy/legal review, and threat modeling.

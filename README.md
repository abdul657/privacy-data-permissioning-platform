# Privacy Data Permissioning Platform

A Go + PostgreSQL portfolio project demonstrating tenant-scoped consent records, purpose-specific authorization checks, and transactional audit events. It is a backend engineering exercise around explicit permission scopes, fail-closed decisions, data integrity, and traceable changes.

> Portfolio/demo project only. This is not a certified privacy, consent, or regulatory-compliance product. Tenant and actor headers are demo selectors, not authentication. Use synthetic data only. See docs/SECURITY.md.

## What it demonstrates

- Go HTTP services using the standard library and PostgreSQL through pgx.
- Tenant-scoped database queries for consent records and audit events.
- Consent grant/re-grant and revocation workflows.
- Exact-match authorization across tenant, subject, purpose, and data category; unknown or revoked consent is denied.
- Atomic transactions that persist consent changes and matching audit events together.
- Parameterized SQL, request validation, bounded JSON bodies, and graceful HTTP shutdown.
- Docker Compose, a non-root container, unit tests, PostgreSQL integration tests, and GitHub Actions CI.

## Quick start

Requirements: Docker Desktop or Docker Engine with Compose.

```bash
git clone https://github.com/abdul657/privacy-data-permissioning-platform.git
cd privacy-data-permissioning-platform
docker compose up --build
```

The API listens on http://localhost:8080. PostgreSQL data is stored in a named volume. SQL migrations are embedded in the Go binary and applied at startup.

Check health:

```bash
curl http://localhost:8080/healthz
```

Run a grant -> authorization check -> revoke -> denial -> audit walkthrough:

```bash
bash scripts/demo.sh
```

## API

All /v1 routes require X-Tenant-ID. X-Actor-ID is optional and defaults to system for audit attribution. These headers are not authentication.

| Method | Path | Description |
|---|---|---|
| GET | /healthz | Liveness check |
| POST | /v1/consents | Grant or re-grant scoped consent |
| GET | /v1/consents?subject_id=... | List tenant-scoped consent records |
| POST | /v1/consents/{id}/revoke | Revoke a consent |
| POST | /v1/authorization/check | Check exact-purpose permission |
| GET | /v1/audit-events?limit=50 | List audit events |

## Run locally

Requirements: Go 1.23+ and PostgreSQL 16+.

```bash
go mod tidy
export DATABASE_URL="postgres://permissioning:permissioning_dev@localhost:5432/permissioning?sslmode=disable"
go run ./cmd/api
```

PowerShell:

```powershell
go mod tidy
$env:DATABASE_URL = 'postgres://permissioning:permissioning_dev@localhost:5432/permissioning?sslmode=disable'
go run ./cmd/api
```

## Testing

```bash
go mod tidy
go test ./...
go vet ./...
```

The PostgreSQL integration test is skipped unless TEST_DATABASE_URL is configured. Set TEST_DATABASE_URL to a test database connection string, then run:

```bash
go test ./internal/store -run TestPostgresConsentLifecycleAndTenantIsolation -v
```

GitHub Actions runs formatting checks, go vet, unit tests, the PostgreSQL integration test, race detection, coverage, and a build against PostgreSQL 16.

## Architecture and security

See docs/ARCHITECTURE.md for the data model and endpoints. See docs/SECURITY.md for implemented safeguards and important production limitations.

## Next steps

Potential extensions include signed identity integration, tenant provisioning, policy versioning, webhook delivery, retention/deletion workflows, an outbox for reliable downstream integrations, rate limiting, OpenTelemetry instrumentation, and deployment infrastructure.

## License

MIT. See LICENSE.

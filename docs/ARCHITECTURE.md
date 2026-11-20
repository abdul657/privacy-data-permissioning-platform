# Architecture

Client / trusted API gateway
          |
          | X-Tenant-ID (demo selector; not an identity credential)
          v
     Go HTTP API
       |       +--> Authorization check (fail closed)
       v
   PostgreSQL
      |     |
      v     v
  Consents  Audit events

## Core model

A consent record is uniquely scoped by (tenant_id, subject_id, purpose, data_category). Granting an existing scope updates it to granted; revocation changes it to revoked. Every grant or revoke writes an audit event in the same transaction.

An authorization decision is allowed only when an exact tenant, subject, purpose, and data-category match has status=granted. No wildcard or inheritance behavior is implemented.

## API

All /v1/* routes require X-Tenant-ID. X-Actor-ID is optional and defaults to system for audit attribution. Both headers are demo selectors only; see SECURITY.md.

| Method | Path | Description |
|---|---|---|
| GET | /healthz | Liveness check |
| POST | /v1/consents | Grant or re-grant scoped consent |
| GET | /v1/consents?subject_id=... | List tenant-scoped consent records |
| POST | /v1/consents/{id}/revoke | Revoke a consent |
| POST | /v1/authorization/check | Exact-purpose authorization check |
| GET | /v1/audit-events?limit=50 | List tenant audit events |

Grant request JSON: {"subject_id":"subject-001","purpose":"product_analytics","data_category":"usage_events","source":"web-preferences"}

Authorization request JSON: {"subject_id":"subject-001","purpose":"product_analytics","data_category":"usage_events"}

Allowed response: {"allowed":true,"reason":"active_consent"}

Missing or revoked consent returns allowed=false.

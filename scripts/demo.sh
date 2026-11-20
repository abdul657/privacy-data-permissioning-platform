#!/usr/bin/env bash
set -euo pipefail
BASE_URL="http://localhost:8080"
TENANT_ID="demo-brand"
SUBJECT_ID="demo-subject-001"

printf '\n1) Create or grant consent\n'
CONSENT=$(curl --fail-with-body -sS -X POST "$BASE_URL/v1/consents" \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT_ID" -H "X-Actor-ID: demo-admin" \
  -d "{\"subject_id\":\"$SUBJECT_ID\",\"purpose\":\"product_analytics\",\"data_category\":\"usage_events\",\"source\":\"demo\"}")
printf '%s\n' "$CONSENT"
CONSENT_ID=$(printf '%s' "$CONSENT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')

printf '\n2) Check permission (expected allowed=true)\n'
curl --fail-with-body -sS -X POST "$BASE_URL/v1/authorization/check" \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"subject_id\":\"$SUBJECT_ID\",\"purpose\":\"product_analytics\",\"data_category\":\"usage_events\"}"

printf '\n\n3) Revoke consent\n'
curl --fail-with-body -sS -X POST "$BASE_URL/v1/consents/$CONSENT_ID/revoke" \
  -H "X-Tenant-ID: $TENANT_ID" -H "X-Actor-ID: demo-admin"

printf '\n\n4) Check permission again (expected allowed=false)\n'
curl --fail-with-body -sS -X POST "$BASE_URL/v1/authorization/check" \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"subject_id\":\"$SUBJECT_ID\",\"purpose\":\"product_analytics\",\"data_category\":\"usage_events\"}"

printf '\n\n5) Audit events\n'
curl --fail-with-body -sS "$BASE_URL/v1/audit-events?limit=10" -H "X-Tenant-ID: $TENANT_ID"
printf '\n'

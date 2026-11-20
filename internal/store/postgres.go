package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abdul657/privacy-data-permissioning-platform/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (p *Postgres) UpsertConsent(ctx context.Context, consent domain.Consent, actorID string) (domain.Consent, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil { return domain.Consent{}, fmt.Errorf("begin consent transaction: %w", err) }
	defer tx.Rollback(ctx)
	const query = `
		INSERT INTO consent_records (tenant_id, subject_id, purpose, data_category, status, source)
		VALUES ($1, $2, $3, $4, 'granted', $5)
		ON CONFLICT (tenant_id, subject_id, purpose, data_category)
		DO UPDATE SET status = 'granted', source = EXCLUDED.source, updated_at = NOW()
		RETURNING id, tenant_id, subject_id, purpose, data_category, status, source, created_at, updated_at`
	var result domain.Consent
	err = tx.QueryRow(ctx, query, consent.TenantID, consent.SubjectID, consent.Purpose, consent.DataCategory, consent.Source).
		Scan(&result.ID, &result.TenantID, &result.SubjectID, &result.Purpose, &result.DataCategory, &result.Status, &result.Source, &result.CreatedAt, &result.UpdatedAt)
	if err != nil { return domain.Consent{}, fmt.Errorf("upsert consent: %w", err) }
	details, _ := json.Marshal(map[string]string{"subject_id": result.SubjectID, "purpose": result.Purpose, "data_category": result.DataCategory, "status": result.Status})
	if err := insertAudit(ctx, tx, result.TenantID, actorID, "consent.granted", "consent", result.ID, details); err != nil { return domain.Consent{}, err }
	if err := tx.Commit(ctx); err != nil { return domain.Consent{}, fmt.Errorf("commit consent transaction: %w", err) }
	return result, nil
}

func (p *Postgres) ListConsents(ctx context.Context, tenantID, subjectID string) ([]domain.Consent, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, tenant_id, subject_id, purpose, data_category, status, source, created_at, updated_at
		FROM consent_records WHERE tenant_id = $1 AND ($2 = '' OR subject_id = $2)
		ORDER BY updated_at DESC LIMIT 500`, tenantID, subjectID)
	if err != nil { return nil, fmt.Errorf("list consents: %w", err) }
	defer rows.Close()
	items := make([]domain.Consent, 0)
	for rows.Next() {
		var item domain.Consent
		if err := rows.Scan(&item.ID, &item.TenantID, &item.SubjectID, &item.Purpose, &item.DataCategory, &item.Status, &item.Source, &item.CreatedAt, &item.UpdatedAt); err != nil { return nil, fmt.Errorf("scan consent: %w", err) }
		items = append(items, item)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate consents: %w", err) }
	return items, nil
}

func (p *Postgres) RevokeConsent(ctx context.Context, tenantID, consentID, actorID string) (domain.Consent, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil { return domain.Consent{}, fmt.Errorf("begin revoke transaction: %w", err) }
	defer tx.Rollback(ctx)
	var result domain.Consent
	err = tx.QueryRow(ctx, `
		UPDATE consent_records SET status = 'revoked', updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, subject_id, purpose, data_category, status, source, created_at, updated_at`, tenantID, consentID).
		Scan(&result.ID, &result.TenantID, &result.SubjectID, &result.Purpose, &result.DataCategory, &result.Status, &result.Source, &result.CreatedAt, &result.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return domain.Consent{}, ErrNotFound }
	if err != nil { return domain.Consent{}, fmt.Errorf("revoke consent: %w", err) }
	details, _ := json.Marshal(map[string]string{"subject_id": result.SubjectID, "purpose": result.Purpose, "data_category": result.DataCategory, "status": result.Status})
	if err := insertAudit(ctx, tx, tenantID, actorID, "consent.revoked", "consent", result.ID, details); err != nil { return domain.Consent{}, err }
	if err := tx.Commit(ctx); err != nil { return domain.Consent{}, fmt.Errorf("commit revoke transaction: %w", err) }
	return result, nil
}

func (p *Postgres) CheckPermission(ctx context.Context, tenantID, subjectID, purpose, dataCategory string) (bool, error) {
	var allowed bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM consent_records WHERE tenant_id = $1 AND subject_id = $2
		AND purpose = $3 AND data_category = $4 AND status = 'granted'
	)`, tenantID, subjectID, purpose, dataCategory).Scan(&allowed)
	if err != nil { return false, fmt.Errorf("check permission: %w", err) }
	return allowed, nil
}

func (p *Postgres) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, tenant_id, actor_id, action, resource_type, resource_id, details, created_at
		FROM audit_events WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil { return nil, fmt.Errorf("list audit events: %w", err) }
	defer rows.Close()
	items := make([]domain.AuditEvent, 0)
	for rows.Next() {
		var item domain.AuditEvent
		var details []byte
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ActorID, &item.Action, &item.ResourceType, &item.ResourceID, &details, &item.CreatedAt); err != nil { return nil, fmt.Errorf("scan audit event: %w", err) }
		if err := json.Unmarshal(details, &item.Details); err != nil { return nil, fmt.Errorf("decode audit event details: %w", err) }
		items = append(items, item)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate audit events: %w", err) }
	return items, nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, tenantID, actorID, action, resourceType, resourceID string, details []byte) error {
	_, err := tx.Exec(ctx, `INSERT INTO audit_events (tenant_id, actor_id, action, resource_type, resource_id, details)
		VALUES ($1, $2, $3, $4, $5, $6)`, tenantID, actorID, action, resourceType, resourceID, details)
	if err != nil { return fmt.Errorf("write audit event: %w", err) }
	return nil
}

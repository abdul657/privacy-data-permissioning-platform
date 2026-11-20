package store

import (
	"context"
	"errors"

	"github.com/abdul657/privacy-data-permissioning-platform/internal/domain"
)

var ErrNotFound = errors.New("record not found")

type Store interface {
	UpsertConsent(context.Context, domain.Consent, string) (domain.Consent, error)
	ListConsents(context.Context, string, string) ([]domain.Consent, error)
	RevokeConsent(context.Context, string, string, string) (domain.Consent, error)
	CheckPermission(context.Context, string, string, string, string) (bool, error)
	ListAuditEvents(context.Context, string, int) ([]domain.AuditEvent, error)
}

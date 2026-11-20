package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/abdul657/privacy-data-permissioning-platform/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresConsentLifecycleAndTenantIsolation(t *testing.T) {
	databaseURL:=os.Getenv("TEST_DATABASE_URL")
	if databaseURL=="" {t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration test")}
	ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second); defer cancel()
	pool,err:=pgxpool.New(ctx,databaseURL); if err!=nil {t.Fatal(err)}; defer pool.Close()
	if err:=Migrate(ctx,pool);err!=nil {t.Fatal(err)}
	repo:=NewPostgres(pool)
	tenantA:=fmt.Sprintf("test-a-%d",time.Now().UnixNano()); tenantB:=tenantA+"-other"
	created,err:=repo.UpsertConsent(ctx,domain.Consent{TenantID:tenantA,SubjectID:"subject-1",Purpose:"analytics",DataCategory:"usage",Source:"integration-test"},"tester")
	if err!=nil {t.Fatal(err)}
	allowed,err:=repo.CheckPermission(ctx,tenantA,"subject-1","analytics","usage")
	if err!=nil||!allowed {t.Fatalf("expected active consent: allowed=%v err=%v",allowed,err)}
	otherAllowed,err:=repo.CheckPermission(ctx,tenantB,"subject-1","analytics","usage")
	if err!=nil||otherAllowed {t.Fatalf("tenant isolation failed: allowed=%v err=%v",otherAllowed,err)}
	items,err:=repo.ListConsents(ctx,tenantB,""); if err!=nil {t.Fatal(err)}
	if len(items)!=0 {t.Fatalf("other tenant saw %d records",len(items))}
	if _,err:=repo.RevokeConsent(ctx,tenantB,created.ID,"tester");!errors.Is(err,ErrNotFound) {t.Fatalf("expected scoped not found, got %v",err)}
	if _,err:=repo.RevokeConsent(ctx,tenantA,created.ID,"tester");err!=nil {t.Fatal(err)}
	allowed,err=repo.CheckPermission(ctx,tenantA,"subject-1","analytics","usage")
	if err!=nil||allowed {t.Fatalf("expected revoked consent to deny: allowed=%v err=%v",allowed,err)}
	events,err:=repo.ListAuditEvents(ctx,tenantA,20); if err!=nil {t.Fatal(err)}
	if len(events)<2 {t.Fatalf("expected grant and revoke events, got %d",len(events))}
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abdul657/privacy-data-permissioning-platform/internal/domain"
	"github.com/abdul657/privacy-data-permissioning-platform/internal/store"
)

type fakeStore struct {
	lastTenant string
	allowed bool
	listErr error
	revokeErr error
}
func (f *fakeStore) UpsertConsent(_ context.Context,c domain.Consent,_ string)(domain.Consent,error) {
	f.lastTenant=c.TenantID; c.ID="consent-1"; c.Status=domain.ConsentGranted; return c,nil
}
func (f *fakeStore) ListConsents(_ context.Context,tenant,subject string)([]domain.Consent,error) { f.lastTenant=tenant; return []domain.Consent{},f.listErr }
func (f *fakeStore) RevokeConsent(_ context.Context,tenant,id,actor string)(domain.Consent,error) {
	f.lastTenant=tenant; if f.revokeErr!=nil{return domain.Consent{},f.revokeErr}; return domain.Consent{ID:id,TenantID:tenant,Status:domain.ConsentRevoked},nil
}
func (f *fakeStore) CheckPermission(_ context.Context,tenant,subject,purpose,category string)(bool,error) { f.lastTenant=tenant; return f.allowed,nil }
func (f *fakeStore) ListAuditEvents(_ context.Context,tenant string,limit int)([]domain.AuditEvent,error) { f.lastTenant=tenant; return []domain.AuditEvent{},nil }

func newTestHandler(s store.Store) http.Handler { return NewHandler(s,slog.New(slog.NewTextHandler(io.Discard,nil))) }
func request(h http.Handler,method,path,body,tenant string)*httptest.ResponseRecorder {
	req:=httptest.NewRequest(method,path,strings.NewReader(body)); if tenant!="" {req.Header.Set("X-Tenant-ID",tenant)}
	req.Header.Set("X-Actor-ID","test-user"); rec:=httptest.NewRecorder(); h.ServeHTTP(rec,req); return rec
}
func TestHealthDoesNotRequireTenant(t *testing.T) {
	rec:=request(newTestHandler(&fakeStore{}),http.MethodGet,"/healthz","","")
	if rec.Code!=http.StatusOK {t.Fatalf("status=%d",rec.Code)}
}
func TestAPIRoutesRequireTenant(t *testing.T) {
	rec:=request(newTestHandler(&fakeStore{}),http.MethodGet,"/v1/consents","","")
	if rec.Code!=http.StatusBadRequest {t.Fatalf("status=%d",rec.Code)}
}
func TestCreateConsentUsesTenantHeader(t *testing.T) {
	stub:=&fakeStore{}
	rec:=request(newTestHandler(stub),http.MethodPost,"/v1/consents",`{"subject_id":"user-42","purpose":"analytics","data_category":"usage"}`,"tenant-a")
	if rec.Code!=http.StatusCreated {t.Fatalf("status=%d body=%s",rec.Code,rec.Body.String())}
	if stub.lastTenant!="tenant-a" {t.Fatalf("tenant=%q",stub.lastTenant)}
	var got domain.Consent
	if err:=json.Unmarshal(rec.Body.Bytes(),&got);err!=nil {t.Fatal(err)}
	if got.Status!=domain.ConsentGranted {t.Fatalf("status=%q",got.Status)}
}
func TestCreateConsentRejectsUnknownFields(t *testing.T) {
	rec:=request(newTestHandler(&fakeStore{}),http.MethodPost,"/v1/consents",`{"subject_id":"u1","purpose":"p","data_category":"c","tenant_id":"tenant-b"}`,"tenant-a")
	if rec.Code!=http.StatusBadRequest {t.Fatalf("status=%d",rec.Code)}
}
func TestAuthorizationDecisionFailsClosed(t *testing.T) {
	rec:=request(newTestHandler(&fakeStore{allowed:false}),http.MethodPost,"/v1/authorization/check",`{"subject_id":"u1","purpose":"analytics","data_category":"usage"}`,"tenant-a")
	if rec.Code!=http.StatusOK||!strings.Contains(rec.Body.String(),`"allowed":false`) {t.Fatalf("response=%d %s",rec.Code,rec.Body.String())}
}
func TestRevokeNotFound(t *testing.T) {
	rec:=request(newTestHandler(&fakeStore{revokeErr:store.ErrNotFound}),http.MethodPost,"/v1/consents/missing/revoke","","tenant-a")
	if rec.Code!=http.StatusNotFound {t.Fatalf("status=%d",rec.Code)}
}
func TestListStoreErrorDoesNotLeakDetails(t *testing.T) {
	rec:=request(newTestHandler(&fakeStore{listErr:errors.New("sensitive database detail")}),http.MethodGet,"/v1/consents","","tenant-a")
	if rec.Code!=http.StatusInternalServerError||strings.Contains(rec.Body.String(),"sensitive") {t.Fatalf("response=%d %s",rec.Code,rec.Body.String())}
}

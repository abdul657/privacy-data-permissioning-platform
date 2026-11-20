package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/abdul657/privacy-data-permissioning-platform/internal/domain"
	"github.com/abdul657/privacy-data-permissioning-platform/internal/store"
)

const maxBodyBytes = 1 << 20
var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:@/-]{0,127}$`)
type contextKey string
const (
	tenantContextKey contextKey = "tenant_id"
	actorContextKey contextKey = "actor_id"
)
type Server struct { store store.Store; logger *slog.Logger }

func NewHandler(dataStore store.Store, logger *slog.Logger) http.Handler {
	if logger == nil { logger = slog.Default() }
	s := &Server{store:dataStore, logger:logger}
	api := http.NewServeMux()
	api.HandleFunc("POST /v1/consents", s.createConsent)
	api.HandleFunc("GET /v1/consents", s.listConsents)
	api.HandleFunc("POST /v1/consents/{id}/revoke", s.revokeConsent)
	api.HandleFunc("POST /v1/authorization/check", s.checkAuthorization)
	api.HandleFunc("GET /v1/audit-events", s.listAuditEvents)
	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w,http.StatusOK,map[string]string{"status":"ok"}) })
	root.Handle("/v1/", tenantMiddleware(api))
	return root
}

func tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		tenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if !identifierPattern.MatchString(tenant) { writeError(w,http.StatusBadRequest,"X-Tenant-ID header is required and must be a valid identifier"); return }
		actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
		if actor == "" { actor = "system" }
		if len(actor)>128 { writeError(w,http.StatusBadRequest,"X-Actor-ID is too long"); return }
		ctx:=context.WithValue(r.Context(),tenantContextKey,tenant)
		ctx=context.WithValue(ctx,actorContextKey,actor)
		next.ServeHTTP(w,r.WithContext(ctx))
	})
}

func (s *Server) createConsent(w http.ResponseWriter,r *http.Request) {
	var req struct {
		SubjectID string `json:"subject_id"`
		Purpose string `json:"purpose"`
		DataCategory string `json:"data_category"`
		Source string `json:"source"`
	}
	if err:=decodeJSON(w,r,&req); err!=nil { writeError(w,http.StatusBadRequest,err.Error()); return }
	req.SubjectID=strings.TrimSpace(req.SubjectID); req.Purpose=strings.TrimSpace(req.Purpose)
	req.DataCategory=strings.TrimSpace(req.DataCategory); req.Source=strings.TrimSpace(req.Source)
	if !validText(req.SubjectID,200)||!validText(req.Purpose,100)||!validText(req.DataCategory,100) { writeError(w,http.StatusBadRequest,"subject_id, purpose, and data_category are required"); return }
	if req.Source=="" { req.Source="api" }
	if !validText(req.Source,100) { writeError(w,http.StatusBadRequest,"source is invalid"); return }
	consent,err:=s.store.UpsertConsent(r.Context(),domain.Consent{TenantID:tenantID(r.Context()),SubjectID:req.SubjectID,Purpose:req.Purpose,DataCategory:req.DataCategory,Source:req.Source},actorID(r.Context()))
	if err!=nil { s.internalError(w,r,"create consent",err); return }
	writeJSON(w,http.StatusCreated,consent)
}
func (s *Server) listConsents(w http.ResponseWriter,r *http.Request) {
	subject:=strings.TrimSpace(r.URL.Query().Get("subject_id"))
	if subject!=""&&!validText(subject,200) { writeError(w,http.StatusBadRequest,"subject_id is invalid"); return }
	items,err:=s.store.ListConsents(r.Context(),tenantID(r.Context()),subject)
	if err!=nil { s.internalError(w,r,"list consents",err); return }
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"count":len(items)})
}
func (s *Server) revokeConsent(w http.ResponseWriter,r *http.Request) {
	id:=strings.TrimSpace(r.PathValue("id"))
	if !identifierPattern.MatchString(id) { writeError(w,http.StatusBadRequest,"consent id is invalid"); return }
	item,err:=s.store.RevokeConsent(r.Context(),tenantID(r.Context()),id,actorID(r.Context()))
	if errors.Is(err,store.ErrNotFound) { writeError(w,http.StatusNotFound,"consent not found"); return }
	if err!=nil { s.internalError(w,r,"revoke consent",err); return }
	writeJSON(w,http.StatusOK,item)
}
func (s *Server) checkAuthorization(w http.ResponseWriter,r *http.Request) {
	var req struct {
		SubjectID string `json:"subject_id"`
		Purpose string `json:"purpose"`
		DataCategory string `json:"data_category"`
	}
	if err:=decodeJSON(w,r,&req); err!=nil { writeError(w,http.StatusBadRequest,err.Error()); return }
	req.SubjectID=strings.TrimSpace(req.SubjectID); req.Purpose=strings.TrimSpace(req.Purpose); req.DataCategory=strings.TrimSpace(req.DataCategory)
	if !validText(req.SubjectID,200)||!validText(req.Purpose,100)||!validText(req.DataCategory,100) { writeError(w,http.StatusBadRequest,"subject_id, purpose, and data_category are required"); return }
	allowed,err:=s.store.CheckPermission(r.Context(),tenantID(r.Context()),req.SubjectID,req.Purpose,req.DataCategory)
	if err!=nil { s.internalError(w,r,"check authorization",err); return }
	reason:="no_active_consent"; if allowed { reason="active_consent" }
	writeJSON(w,http.StatusOK,map[string]any{"allowed":allowed,"reason":reason})
}
func (s *Server) listAuditEvents(w http.ResponseWriter,r *http.Request) {
	limit:=50
	if raw:=r.URL.Query().Get("limit"); raw!="" {
		value,err:=strconv.Atoi(raw); if err!=nil||value<1||value>100 { writeError(w,http.StatusBadRequest,"limit must be between 1 and 100"); return }; limit=value
	}
	items,err:=s.store.ListAuditEvents(r.Context(),tenantID(r.Context()),limit)
	if err!=nil { s.internalError(w,r,"list audit events",err); return }
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"count":len(items)})
}
func (s *Server) internalError(w http.ResponseWriter,r *http.Request,operation string,err error) {
	s.logger.Error("request failed","operation",operation,"tenant_id",tenantID(r.Context()),"error",err)
	writeError(w,http.StatusInternalServerError,"internal server error")
}
func decodeJSON(w http.ResponseWriter,r *http.Request,destination any) error {
	r.Body=http.MaxBytesReader(w,r.Body,maxBodyBytes)
	decoder:=json.NewDecoder(r.Body); decoder.DisallowUnknownFields()
	if err:=decoder.Decode(destination); err!=nil { return errors.New("request body must be valid JSON with supported fields") }
	if err:=decoder.Decode(&struct{}{}); !errors.Is(err,io.EOF) { return errors.New("request body must contain one JSON object") }
	return nil
}
func validText(value string,max int) bool {
	if value==""||len(value)>max { return false }
	for _,ch:=range value { if ch<0x20||ch==0x7f { return false } }
	return true
}
func tenantID(ctx context.Context) string { value,_:=ctx.Value(tenantContextKey).(string); return value }
func actorID(ctx context.Context) string { value,_:=ctx.Value(actorContextKey).(string); return value }
func writeError(w http.ResponseWriter,status int,message string) { writeJSON(w,status,map[string]string{"error":message}) }
func writeJSON(w http.ResponseWriter,status int,value any) {
	w.Header().Set("Content-Type","application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

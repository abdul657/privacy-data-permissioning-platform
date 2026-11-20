package domain

import "time"

const (
	ConsentGranted = "granted"
	ConsentRevoked = "revoked"
)

type Consent struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	SubjectID    string    `json:"subject_id"`
	Purpose      string    `json:"purpose"`
	DataCategory string    `json:"data_category"`
	Status       string    `json:"status"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AuditEvent struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	ActorID      string            `json:"actor_id"`
	Action       string            `json:"action"`
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Details      map[string]string `json:"details"`
	CreatedAt    time.Time         `json:"created_at"`
}

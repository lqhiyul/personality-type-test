package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type AdminAuditStore struct {
	db  *sql.DB
	now func() time.Time
}

type AdminAuditEntry struct {
	Action     string
	TargetType string
	TargetID   string
	IP         string
	UserAgent  string
}

func NewAdminAuditStore(db *sql.DB) *AdminAuditStore {
	return &AdminAuditStore{
		db:  db,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *AdminAuditStore) Log(ctx context.Context, entry AdminAuditEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	action := strings.TrimSpace(entry.Action)
	if action == "" {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_audit_logs (action, target_type, target_id, ip, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, action, nullableAuditString(entry.TargetType), nullableAuditString(entry.TargetID), nullableAuditString(entry.IP), nullableAuditString(entry.UserAgent), s.now().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert admin audit log: %w", err)
	}
	return nil
}

func (a *App) recordAdminAudit(r *http.Request, action, targetType, targetID string) {
	if a == nil {
		return
	}
	err := a.auditStore.Log(r.Context(), AdminAuditEntry{
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IP:         a.clientIP(r),
		UserAgent:  truncateAuditValue(r.UserAgent(), 512),
	})
	if err != nil {
		log.Printf("admin audit log error: %v", err)
	}
}

func nullableAuditString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func truncateAuditValue(value string, maxBytes int) string {
	value = strings.TrimSpace(value)
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}

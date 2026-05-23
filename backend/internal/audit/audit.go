// Package audit writes append-only entries into the audit_logs table.
//
// Every privileged action (approve user, change role, publish case,
// delete attachment, …) should call audit.Write through the request
// scope. The function never blocks on caller error; on failure it logs
// and returns nil, so an audit failure never hides a successful action.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Action is a stable identifier for an event type. Use lowercase dotted
// names so we can filter by prefix in the admin UI later.
type Action string

const (
	ActionUserRegister     Action = "user.register"
	ActionUserApprove      Action = "user.approve"
	ActionUserReject       Action = "user.reject"
	ActionUserSuspend      Action = "user.suspend"
	ActionUserReactivate   Action = "user.reactivate"
	ActionUserRoleChange   Action = "user.role_change"
	ActionUserLogin        Action = "user.login"
	ActionUserLogout       Action = "user.logout"
	ActionUserPasswordSet  Action = "user.password_change"

	ActionCaseCreate    Action = "case.create"
	ActionCaseUpdate    Action = "case.update"
	ActionCaseSubmit    Action = "case.submit"
	ActionCaseAssign    Action = "case.assign"
	ActionCaseVerify    Action = "case.verify"
	ActionCasePublish   Action = "case.publish"
	ActionCaseUnpublish Action = "case.unpublish"
	ActionCaseDelete    Action = "case.delete"

	ActionPersonCreate Action = "person.create"
	ActionPersonUpdate Action = "person.update"
	ActionPersonMerge  Action = "person.merge"

	ActionAttachmentUpload Action = "attachment.upload"
	ActionAttachmentDelete Action = "attachment.delete"
)

// Querier is the minimal interface we need: anything that can execute
// SQL with placeholder args. Both *pgxpool.Pool and pgx.Tx satisfy it.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Entry describes a single audit row.
type Entry struct {
	UserID     *uuid.UUID
	Action     Action
	TargetType string
	TargetID   string
	Metadata   map[string]any
	IP         string
	UserAgent  string
}

// Write inserts the entry. Errors are logged but not returned, so callers
// don't need to handle them — auditing should never break a request.
func Write(ctx context.Context, q Querier, e Entry, logger *slog.Logger) {
	var meta []byte
	if len(e.Metadata) > 0 {
		var err error
		meta, err = json.Marshal(e.Metadata)
		if err != nil {
			if logger != nil {
				logger.Warn("audit: marshal metadata", "err", err)
			}
		}
	}

	const stmt = `
INSERT INTO audit_logs (user_id, action, target_type, target_id, metadata, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::inet, $7)
`
	_, err := q.Exec(ctx, stmt,
		e.UserID, string(e.Action), nullIfEmpty(e.TargetType), nullIfEmpty(e.TargetID),
		meta, e.IP, e.UserAgent,
	)
	if err != nil && logger != nil {
		logger.Warn("audit: insert failed",
			"action", e.Action, "target_type", e.TargetType, "target_id", e.TargetID, "err", err,
		)
	}
}

// FromRequest extracts IP and User-Agent out of an http.Request.
// It honours X-Forwarded-For only when set by a trusted reverse proxy
// (chi RealIP middleware already does that for us).
func FromRequest(r *http.Request) (ip, userAgent string) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host, r.UserAgent()
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

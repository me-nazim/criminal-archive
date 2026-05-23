package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// Error is the canonical API error envelope. It implements `error` so
// service-layer code can `return httpx.NotFound(...)` and the handler
// can do a single `httpx.WriteError(w, r, err)`.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Constructors -------------------------------------------------------------

func BadRequest(msg string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

func ValidationError(fields map[string]string, msg string) *Error {
	if msg == "" {
		msg = "Validation failed."
	}
	return &Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    "validation_error",
		Message: msg,
		Fields:  fields,
	}
}

func Unauthenticated(msg string) *Error {
	if msg == "" {
		msg = "Authentication required."
	}
	return &Error{Status: http.StatusUnauthorized, Code: "unauthenticated", Message: msg}
}

func TokenExpired() *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "token_expired", Message: "Access token has expired."}
}

func Forbidden(code, msg string) *Error {
	if code == "" {
		code = "forbidden"
	}
	if msg == "" {
		msg = "You do not have permission to perform this action."
	}
	return &Error{Status: http.StatusForbidden, Code: code, Message: msg}
}

func NotFound(msg string) *Error {
	if msg == "" {
		msg = "Resource not found."
	}
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

func Conflict(code, msg string) *Error {
	if code == "" {
		code = "conflict"
	}
	return &Error{Status: http.StatusConflict, Code: code, Message: msg}
}

func StateInvalid(msg string) *Error {
	return &Error{Status: http.StatusConflict, Code: "state_transition_invalid", Message: msg}
}

func RateLimited(msg string) *Error {
	if msg == "" {
		msg = "Too many requests."
	}
	return &Error{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: msg}
}

func Internal(msg string) *Error {
	if msg == "" {
		msg = "Internal server error."
	}
	return &Error{Status: http.StatusInternalServerError, Code: "internal_error", Message: msg}
}

// WriteError serialises err into the canonical error envelope. Anything
// that is not a *Error becomes a 500 with code internal_error and a
// generic message — the original error is logged but not exposed.
func WriteError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	rid := middleware.GetReqID(r.Context())

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		apiErr = Internal("")
		if logger != nil {
			logger.Error("internal error", "err", err, "request_id", rid, "path", r.URL.Path)
		}
	}

	envelope := map[string]any{
		"error": map[string]any{
			"code":       apiErr.Code,
			"message":    apiErr.Message,
			"request_id": rid,
		},
	}
	if len(apiErr.Fields) > 0 {
		envelope["error"].(map[string]any)["fields"] = apiErr.Fields
	}

	WriteJSON(w, apiErr.Status, envelope)
}

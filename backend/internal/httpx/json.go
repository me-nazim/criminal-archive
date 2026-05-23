// Package httpx contains small HTTP helpers used across handlers:
// consistent JSON encoding, structured error responses, and request
// decoding with reasonable defaults.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// MaxBodyBytes caps inbound JSON request bodies to a sane size; large
// uploads go directly to object storage, never through the API.
const MaxBodyBytes = 1 << 20 // 1 MiB

// WriteJSON encodes payload as JSON with the given status code.
// It silently ignores write errors (the client has already disconnected).
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

// NoContent writes a 204 with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// DecodeJSON reads a JSON body into dst. It returns a *Error suitable for
// passing straight to WriteError.
func DecodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		switch {
		case errors.As(err, &syntaxErr):
			return BadRequest("invalid JSON: " + err.Error())
		case errors.Is(err, io.EOF):
			return BadRequest("empty request body")
		case errors.Is(err, io.ErrUnexpectedEOF):
			return BadRequest("request body ended unexpectedly")
		default:
			return BadRequest(fmt.Sprintf("invalid request: %s", err))
		}
	}
	if dec.More() {
		return BadRequest("request body must contain a single JSON object")
	}
	return nil
}

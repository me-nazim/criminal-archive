package settings

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Handlers exposes admin-only setting CRUD plus a public branding endpoint.
type Handlers struct {
	store  *Store
	logger *slog.Logger
	tester ProviderTester
}

// ProviderTester is implemented by the email manager (test send) and the
// storage manager (HEAD/PUT round-trip) so the admin UI can verify config
// without leaving the screen.
type ProviderTester interface {
	TestEmail(r *http.Request, to string, cfg EmailConfig) error
	TestStorage(r *http.Request, cfg StorageConfig) error
}

// NewHandlers constructs Handlers.
func NewHandlers(store *Store, tester ProviderTester, logger *slog.Logger) *Handlers {
	return &Handlers{store: store, tester: tester, logger: logger}
}

// secretPathsByKey is the registry of which JSON paths in each settings
// row contain secrets. Used both for sealing on write and masking on read.
var secretPathsByKey = map[string][]string{
	"email":   EmailSecretPaths,
	"storage": StorageSecretPaths,
}

// MountPublic mounts the unauthenticated /settings/branding endpoint so
// the SPA can render the right title / colours before login.
func (h *Handlers) MountPublic(r chi.Router) {
	r.Get("/settings/branding", h.getBrandingPublic)
	r.Get("/settings/banner", h.getBannerPublic)
}

// MountAdmin mounts the admin-only routes.
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Get("/settings", h.list)
	r.Get("/settings/{key}", h.get)
	r.Put("/settings/{key}", h.put)
	r.Post("/settings/email/test", h.testEmail)
	r.Post("/settings/storage/test", h.testStorage)
}

// ---- public ----------------------------------------------------------

func (h *Handlers) getBrandingPublic(w http.ResponseWriter, r *http.Request) {
	b, err := h.store.GetBranding(r.Context())
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, b)
}

func (h *Handlers) getBannerPublic(w http.ResponseWriter, r *http.Request) {
	f, err := h.store.GetFeatures(r.Context())
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if !f.BannerEnabled && !f.MaintenanceMode {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, f)
}

// ---- admin -----------------------------------------------------------

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	out, err := h.store.listForAdmin(r.Context(), secretPathsByKey)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	cipherConfigured := h.store.cipher != nil
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"data":              out,
		"cipher_configured": cipherConfigured,
	})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	raw, err := h.store.GetRaw(r.Context(), key)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapErr(err))
		return
	}
	if paths := secretPathsByKey[key]; len(paths) > 0 {
		if masked, mErr := maskSecrets(raw, paths); mErr == nil {
			raw = masked
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"key": key, "value": json.RawMessage(raw)})
}

type putReq struct {
	Value json.RawMessage `json:"value"`
}

func (h *Handlers) put(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req putReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if len(req.Value) == 0 {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("value is required"))
		return
	}
	var patch map[string]any
	if err := json.Unmarshal(req.Value, &patch); err != nil {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("value must be a JSON object"))
		return
	}
	id := auth.MustIdentity(r.Context())
	uid := id.UserID
	if err := h.store.Patch(r.Context(), key, patch, secretPathsByKey[key], &uid); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	ip, ua := audit.FromRequest(r)
	audit.Write(r.Context(), h.store.pool, audit.Entry{
		UserID:     &uid,
		Action:     audit.Action("settings.update"),
		TargetType: "settings",
		TargetID:   key,
		IP:         ip, UserAgent: ua,
		Metadata: map[string]any{"keys": flatKeys(patch, secretPathsByKey[key])},
	}, h.logger)
	h.get(w, r)
}

type testEmailReq struct {
	To     string      `json:"to"`
	Config EmailConfig `json:"config"`
}

func (h *Handlers) testEmail(w http.ResponseWriter, r *http.Request) {
	if h.tester == nil {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("email tester not available"))
		return
	}
	var req testEmailReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	cfg, err := h.materialiseEmailConfig(r, req.Config)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.tester.TestEmail(r, req.To, cfg); err != nil {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("test send failed: "+err.Error()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handlers) testStorage(w http.ResponseWriter, r *http.Request) {
	if h.tester == nil {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("storage tester not available"))
		return
	}
	var cfg StorageConfig
	if err := httpx.DecodeJSON(r, &cfg); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	cfg, err := h.materialiseStorageConfig(r, cfg)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.tester.TestStorage(r, cfg); err != nil {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("test failed: "+err.Error()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// materialiseEmailConfig fills empty secret fields from the stored
// (decrypted) config so the operator doesn't have to retype passwords
// to test.
func (h *Handlers) materialiseEmailConfig(r *http.Request, in EmailConfig) (EmailConfig, error) {
	cur, err := h.store.GetEmail(r.Context())
	if err != nil && err != ErrNotFound {
		return in, err
	}
	if in.SMTP.Password == "" {
		in.SMTP.Password = cur.SMTP.Password
	}
	if in.Resend.APIKey == "" {
		in.Resend.APIKey = cur.Resend.APIKey
	}
	if in.ElasticMail.APIKey == "" {
		in.ElasticMail.APIKey = cur.ElasticMail.APIKey
	}
	if in.ElasticMail.BaseURL == "" {
		in.ElasticMail.BaseURL = cur.ElasticMail.BaseURL
	}
	return in, nil
}

func (h *Handlers) materialiseStorageConfig(r *http.Request, in StorageConfig) (StorageConfig, error) {
	cur, err := h.store.GetStorage(r.Context())
	if err != nil && err != ErrNotFound {
		return in, err
	}
	if in.SecretKey == "" {
		in.SecretKey = cur.SecretKey
	}
	return in, nil
}

func mapErr(err error) error {
	if err == ErrNotFound {
		return httpx.NotFound("Setting not found.")
	}
	return err
}

// flatKeys returns the dot-notation paths present in a patch — useful
// for audit metadata without leaking values.
func flatKeys(m map[string]any, secretPaths []string) []string {
	out := []string{}
	var walk func(string, map[string]any)
	walk = func(prefix string, mm map[string]any) {
		for k, v := range mm {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			if isSecret(path, secretPaths) {
				out = append(out, path+"=<redacted>")
				continue
			}
			if sub, ok := v.(map[string]any); ok {
				walk(path, sub)
				continue
			}
			out = append(out, path)
		}
	}
	walk("", m)
	return out
}

func isSecret(path string, paths []string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

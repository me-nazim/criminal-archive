package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store reads and writes app_settings rows. It caches values in memory
// for a short window and supports versioned reloads on update.
type Store struct {
	pool   *pgxpool.Pool
	cipher *Cipher
	logger *slog.Logger

	mu      sync.RWMutex
	cache   map[string]json.RawMessage
	loaded  time.Time
	version uint64 // bumped on every Set so listeners can react
}

// New constructs a Store. cipher may be nil; in that case secret fields
// are stored in plain text and a warning is emitted on every Set.
func New(pool *pgxpool.Pool, cipher *Cipher, logger *slog.Logger) *Store {
	return &Store{pool: pool, cipher: cipher, logger: logger, cache: map[string]json.RawMessage{}}
}

// Cipher returns the underlying cipher (may be nil).
func (s *Store) Cipher() *Cipher { return s.cipher }

// Version returns the current cache version. Subscribers may compare
// versions to decide whether to reload.
func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// Reload forces a database read of every row.
func (s *Store) Reload(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT key, value FROM app_settings`)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k string
		var v []byte
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		out[k] = json.RawMessage(v)
	}
	s.mu.Lock()
	s.cache = out
	s.loaded = time.Now()
	s.version++
	s.mu.Unlock()
	return rows.Err()
}

// GetRaw returns the JSON value for a key, hitting the cache if fresh.
// Encrypted secrets are NOT decrypted here — use Get with a typed
// destination, which delegates to UnsealMap.
func (s *Store) GetRaw(ctx context.Context, key string) (json.RawMessage, error) {
	s.mu.RLock()
	if time.Since(s.loaded) < 30*time.Second {
		if v, ok := s.cache[key]; ok {
			s.mu.RUnlock()
			return v, nil
		}
	}
	s.mu.RUnlock()
	if err := s.Reload(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.cache[key]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

// ErrNotFound is returned by GetRaw when the row does not exist.
var ErrNotFound = errors.New("settings: key not found")

// Get unmarshals the (decrypted) JSON for key into dest.
func (s *Store) Get(ctx context.Context, key string, dest any) error {
	raw, err := s.GetRaw(ctx, key)
	if err != nil {
		return err
	}
	plain, err := s.unsealJSON(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, dest)
}

// Set persists a JSON value, sealing any plain string entries listed in
// secretPaths. updatedBy may be nil.
//
// secretPaths is a list of dot-notation paths within the value JSON, e.g.
//   ["smtp.password", "resend.api_key"]
func (s *Store) Set(ctx context.Context, key string, value any, secretPaths []string, updatedBy *uuid.UUID) error {
	plainBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("settings: marshal: %w", err)
	}
	sealed, err := s.sealJSON(plainBytes, secretPaths)
	if err != nil {
		return err
	}
	const stmt = `
INSERT INTO app_settings (key, value, updated_by, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (key) DO UPDATE
   SET value      = EXCLUDED.value,
       updated_by = EXCLUDED.updated_by,
       updated_at = now()`
	if _, err := s.pool.Exec(ctx, stmt, key, sealed, updatedBy); err != nil {
		return err
	}
	return s.Reload(ctx)
}

// Patch reads the existing value for key, applies a shallow merge of
// patch (preserving sealed secrets when patch leaves a secret empty),
// and writes it back.
func (s *Store) Patch(ctx context.Context, key string, patch map[string]any, secretPaths []string, updatedBy *uuid.UUID) error {
	current := map[string]any{}
	if raw, err := s.GetRaw(ctx, key); err == nil {
		plain, err := s.unsealJSON(raw)
		if err == nil {
			_ = json.Unmarshal(plain, &current)
		}
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	deepMergePreservingSecrets(current, patch, secretPaths, "")
	return s.Set(ctx, key, current, secretPaths, updatedBy)
}

// ---- helpers ---------------------------------------------------------

// sealJSON walks the JSON document and replaces every leaf at a secretPath
// with `{"__enc": true, "ciphertext": "<base64>"}`. Empty strings are kept
// as empty strings so the admin UI can clear a secret deliberately.
func (s *Store) sealJSON(raw []byte, secretPaths []string) ([]byte, error) {
	if len(secretPaths) == 0 {
		return raw, nil
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for _, p := range secretPaths {
		if err := s.transformAt(doc, strings.Split(p, "."), s.sealString); err != nil {
			return nil, err
		}
	}
	return json.Marshal(doc)
}

// unsealJSON walks every nested object and decrypts {"__enc": true, ...}
// entries. Returns the decrypted JSON bytes.
func (s *Store) unsealJSON(raw []byte) ([]byte, error) {
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	doc = s.unsealWalk(doc)
	return json.Marshal(doc)
}

func (s *Store) unsealWalk(node any) any {
	switch v := node.(type) {
	case map[string]any:
		if isSealed(v) {
			ct, _ := v["ciphertext"].(string)
			if ct == "" {
				return ""
			}
			if s.cipher == nil {
				return "" // refuse to expose ciphertext when key is missing
			}
			plain, err := s.cipher.Decrypt(ct)
			if err != nil {
				if s.logger != nil {
					s.logger.Warn("settings: decrypt failed", "err", err)
				}
				return ""
			}
			return plain
		}
		for k, val := range v {
			v[k] = s.unsealWalk(val)
		}
		return v
	case []any:
		for i, val := range v {
			v[i] = s.unsealWalk(val)
		}
		return v
	}
	return node
}

func (s *Store) sealString(v any) (any, error) {
	str, ok := v.(string)
	if !ok || str == "" {
		return v, nil
	}
	if s.cipher == nil {
		// No cipher: keep plaintext but log loudly. Operators can still
		// configure the system, but the admin UI flags it.
		if s.logger != nil {
			s.logger.Warn("settings: cipher not configured; storing secret in plaintext")
		}
		return str, nil
	}
	ct, err := s.cipher.Encrypt(str)
	if err != nil {
		return nil, err
	}
	return map[string]any{"__enc": true, "ciphertext": ct}, nil
}

// transformAt walks `doc` along path[] and applies fn to the leaf. If a
// path segment doesn't exist it is silently ignored — the admin form may
// not include every field on every save.
func (s *Store) transformAt(doc any, path []string, fn func(any) (any, error)) error {
	if len(path) == 0 {
		return nil
	}
	m, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	if len(path) == 1 {
		v, ok := m[path[0]]
		if !ok {
			return nil
		}
		// If the field is already sealed, treat it as "no change" — the
		// admin form sends back the placeholder, not the original.
		if mv, isMap := v.(map[string]any); isMap && isSealed(mv) {
			return nil
		}
		nv, err := fn(v)
		if err != nil {
			return err
		}
		m[path[0]] = nv
		return nil
	}
	return s.transformAt(m[path[0]], path[1:], fn)
}

func isSealed(m map[string]any) bool {
	v, ok := m["__enc"].(bool)
	return ok && v
}

// deepMergePreservingSecrets walks patch into dst. If a secret path is
// present in patch but the value is the empty string, the existing value
// is preserved (so the admin UI can render "•••••" without resending it).
func deepMergePreservingSecrets(dst, patch map[string]any, secretPaths []string, prefix string) {
	secret := func(path string) bool {
		for _, p := range secretPaths {
			if p == path {
				return true
			}
		}
		return false
	}
	for k, v := range patch {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if secret(path) {
			if str, ok := v.(string); ok && str == "" {
				continue // keep existing
			}
			dst[k] = v
			continue
		}
		if pm, ok := v.(map[string]any); ok {
			cur, _ := dst[k].(map[string]any)
			if cur == nil {
				cur = map[string]any{}
			}
			deepMergePreservingSecrets(cur, pm, secretPaths, path)
			dst[k] = cur
			continue
		}
		dst[k] = v
	}
}

// ---- typed accessors -------------------------------------------------

// Branding is the public-facing site identity.
type Branding struct {
	SiteNameBN   string                 `json:"site_name_bn"`
	SiteNameEN   string                 `json:"site_name_en"`
	ShortName    string                 `json:"short_name"`
	TaglineBN    string                 `json:"tagline_bn"`
	TaglineEN    string                 `json:"tagline_en"`
	PrimaryColor string                 `json:"primary_color"`
	AccentColor  string                 `json:"accent_color"`
	LogoURL      string                 `json:"logo_url"`
	FaviconURL   string                 `json:"favicon_url"`
	SupportEmail string                 `json:"support_email"`
	Social       map[string]string      `json:"social"`
	Extras       map[string]any         `json:"extras,omitempty"`
}

// EmailConfig is the active email-provider configuration.
type EmailConfig struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"` // smtp | resend | elastic_mail
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	ReplyTo     string `json:"reply_to,omitempty"`
	SMTP        struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		StartTLS bool   `json:"starttls"`
		UseTLS   bool   `json:"use_tls"`
	} `json:"smtp"`
	Resend struct {
		APIKey string `json:"api_key"`
	} `json:"resend"`
	ElasticMail struct {
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
	} `json:"elastic_mail"`
}

// EmailSecretPaths lists every encrypted leaf within EmailConfig.
var EmailSecretPaths = []string{
	"smtp.password",
	"resend.api_key",
	"elastic_mail.api_key",
}

// StorageConfig is the active object-storage configuration.
type StorageConfig struct {
	Enabled        bool   `json:"enabled"`
	Driver         string `json:"driver"` // r2 | aws_s3 | minio | s3_compatible
	Bucket         string `json:"bucket"`
	Region         string `json:"region"`
	Endpoint       string `json:"endpoint"`
	AccessKey      string `json:"access_key"`
	SecretKey      string `json:"secret_key"`
	PublicBaseURL  string `json:"public_base_url"`
	ForcePathStyle bool   `json:"force_path_style"`
}

// StorageSecretPaths lists every encrypted leaf within StorageConfig.
var StorageSecretPaths = []string{"secret_key"}

// Features is the feature-flag bundle.
type Features struct {
	AllowPublicRegistration  bool   `json:"allow_public_registration"`
	RequireEmailVerification bool   `json:"require_email_verification"`
	MaintenanceMode          bool   `json:"maintenance_mode"`
	MaintenanceMessageBN     string `json:"maintenance_message_bn"`
	MaintenanceMessageEN     string `json:"maintenance_message_en"`
	BannerEnabled            bool   `json:"banner_enabled"`
	BannerLevel              string `json:"banner_level"`
	BannerMessageBN          string `json:"banner_message_bn"`
	BannerMessageEN          string `json:"banner_message_en"`
}

// GetBranding loads the branding row.
func (s *Store) GetBranding(ctx context.Context) (Branding, error) {
	var b Branding
	if err := s.Get(ctx, "branding", &b); err != nil {
		return b, err
	}
	return b, nil
}

// GetEmail loads + decrypts the email row.
func (s *Store) GetEmail(ctx context.Context) (EmailConfig, error) {
	var e EmailConfig
	if err := s.Get(ctx, "email", &e); err != nil {
		return e, err
	}
	return e, nil
}

// GetStorage loads + decrypts the storage row.
func (s *Store) GetStorage(ctx context.Context) (StorageConfig, error) {
	var sc StorageConfig
	if err := s.Get(ctx, "storage", &sc); err != nil {
		return sc, err
	}
	return sc, nil
}

// GetFeatures loads the feature flags.
func (s *Store) GetFeatures(ctx context.Context) (Features, error) {
	var f Features
	if err := s.Get(ctx, "features", &f); err != nil {
		return f, err
	}
	return f, nil
}

// row is a thin marshal helper for the public API; we expose it from
// handlers but not the cache directly so secrets never leak.
type row struct {
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Description *string         `json:"description,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// listForAdmin returns every settings row with secrets masked
// (`{"__masked": true}` placeholders) so the admin UI can render the
// form without ever receiving plaintext secrets.
func (s *Store) listForAdmin(ctx context.Context, secretPathsByKey map[string][]string) ([]row, error) {
	const stmt = `SELECT key, value, description, updated_at FROM app_settings ORDER BY key`
	rows, err := s.pool.Query(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []row{}
	for rows.Next() {
		var r row
		var desc *string
		if err := rows.Scan(&r.Key, &r.Value, &desc, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.Description = desc
		paths := secretPathsByKey[r.Key]
		if len(paths) > 0 {
			masked, err := maskSecrets(r.Value, paths)
			if err == nil {
				r.Value = masked
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func maskSecrets(raw json.RawMessage, secretPaths []string) (json.RawMessage, error) {
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	for _, p := range secretPaths {
		walkAndMask(doc, strings.Split(p, "."))
	}
	return json.Marshal(doc)
}

func walkAndMask(doc any, path []string) {
	m, ok := doc.(map[string]any)
	if !ok {
		return
	}
	if len(path) == 1 {
		if v, ok := m[path[0]]; ok {
			switch vv := v.(type) {
			case map[string]any:
				if isSealed(vv) {
					m[path[0]] = map[string]any{"__masked": true}
				}
			case string:
				if vv != "" {
					m[path[0]] = map[string]any{"__masked": true}
				}
			}
		}
		return
	}
	walkAndMask(m[path[0]], path[1:])
}

// MarshalIfErr is a tiny helper used by the migrate-pglate path; not
// public API.
func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// guard to silence unused-import warnings during refactors
var _ = pgx.ErrNoRows

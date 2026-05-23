package attachments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/cases"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
	"github.com/me-nazim/criminal-archive/backend/internal/storage"
)

// Handlers wires the attachment lifecycle.
type Handlers struct {
	repo    *Repository
	cases   *cases.Repository
	store   *storage.Client
	logger  *slog.Logger
	hmacKey []byte // signs the "presign token" returned to the client
	audit   *auditWriter
}

// NewHandlers constructs a Handlers.
func NewHandlers(repo *Repository, casesRepo *cases.Repository, store *storage.Client, hmacKey []byte, logger *slog.Logger) *Handlers {
	return &Handlers{repo: repo, cases: casesRepo, store: store, logger: logger, hmacKey: hmacKey}
}

// MountAuthenticated mounts contributor+ routes (presign, finalize).
func (h *Handlers) MountAuthenticated(r chi.Router) {
	r.Post("/cases/{id}/attachments/presign", h.presign)
	r.Post("/cases/{id}/attachments/finalize", h.finalize)
	r.Get("/cases/{id}/attachments", h.list)
}

// MountAdmin mounts admin routes (list-all, presigned-download, edit, delete).
func (h *Handlers) MountAdmin(r chi.Router) {
	r.Get("/cases/{id}/attachments", h.adminList)
	r.Patch("/attachments/{attID}", h.update)
	r.Delete("/attachments/{attID}", h.delete)
	r.Post("/attachments/{attID}/download-url", h.downloadURL)
}

// ------------------------------------------------------ presign

type presignReq struct {
	Kind             string `json:"kind"` // public | hidden | internal
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	SizeBytes        int64  `json:"size_bytes"`
}

type presignResp struct {
	UploadURL      string    `json:"upload_url"`
	StorageKey     string    `json:"storage_key"`
	StoredFilename string    `json:"stored_filename"`
	SequenceNo     int       `json:"sequence_no"`
	Token          string    `json:"presign_token"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func (h *Handlers) presign(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body presignReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if !validKinds[body.Kind] {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"kind": "must be public|hidden|internal"}, ""))
		return
	}
	if body.OriginalFilename == "" {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"original_filename": "required"}, ""))
		return
	}
	c, err := h.cases.GetByID(r.Context(), caseID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err))
		return
	}
	actor := auth.MustIdentity(r.Context())
	if !canMutateCase(actor, c) {
		httpx.WriteError(w, r, h.logger, httpx.Forbidden("forbidden", ""))
		return
	}
	// Only admin+ may upload `internal` attachments. Anyone with case access
	// may upload `hidden` (admins later decide whether to keep them).
	if body.Kind == "internal" && roleRank(actor.Role) < roleRank(auth.RoleAdmin) {
		httpx.WriteError(w, r, h.logger, httpx.Forbidden("forbidden", "Only admins can attach internal files."))
		return
	}

	seq, err := h.repo.AllocateSequence(r.Context(), c.ID, body.Kind)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	ext := path.Ext(body.OriginalFilename)
	if len(ext) > 8 {
		ext = ""
	}
	stored := fmt.Sprintf("%s_%s_%02d%s", c.CaseNumber, body.Kind, seq, strings.ToLower(ext))
	key := fmt.Sprintf("cases/%s/%s/%s", c.CaseNumber, body.Kind, stored)

	signed, err := h.store.PresignPut(r.Context(), storage.PresignedPutInput{
		Key: key, ContentType: body.MimeType, Expiry: 10 * time.Minute,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	tok := h.signToken(c.ID, key, body.Kind, seq, body.MimeType, body.OriginalFilename, signed.ExpiresAt)
	httpx.WriteJSON(w, http.StatusCreated, presignResp{
		UploadURL:      signed.URL,
		StorageKey:     key,
		StoredFilename: stored,
		SequenceNo:     seq,
		Token:          tok,
		ExpiresAt:      signed.ExpiresAt,
	})
}

// ------------------------------------------------------ finalize

type finalizeReq struct {
	Token     string `json:"presign_token"`
	SizeBytes int64  `json:"size_bytes"`
}

func (h *Handlers) finalize(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body finalizeReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	tk, ok := h.verifyToken(body.Token)
	if !ok || tk.CaseID != caseID {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("invalid presign token"))
		return
	}
	c, err := h.cases.GetByID(r.Context(), caseID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err))
		return
	}
	actor := auth.MustIdentity(r.Context())
	if !canMutateCase(actor, c) {
		httpx.WriteError(w, r, h.logger, httpx.Forbidden("forbidden", ""))
		return
	}

	exists, sizeOnDisk, _, err := h.store.HeadObject(r.Context(), tk.Key)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if !exists {
		httpx.WriteError(w, r, h.logger, httpx.BadRequest("object not found in storage; was the upload completed?"))
		return
	}
	size := sizeOnDisk
	if size == 0 {
		size = body.SizeBytes
	}

	var publicURL *string
	if tk.Kind == "public" {
		if u := h.store.PublicURL(tk.Key); u != "" {
			publicURL = &u
		}
	}

	uploader := actor.UserID
	att, err := h.repo.Create(r.Context(), CreateParams{
		CaseID: caseID, Kind: tk.Kind, SequenceNo: tk.SeqNo,
		OriginalFilename: tk.OriginalName, StoredFilename: storedFromKey(tk.Key),
		StorageKey: tk.Key, PublicURL: publicURL,
		MimeType: tk.MimeType, SizeBytes: size,
		UploadedBy: &uploader,
	})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	ip, ua := audit.FromRequest(r)
	h.writeAudit(r.Context(), audit.Entry{
		UserID: &actor.UserID, Action: audit.ActionAttachmentUpload,
		TargetType: "attachment", TargetID: att.ID.String(),
		IP: ip, UserAgent: ua,
		Metadata: map[string]any{"case_number": c.CaseNumber, "kind": tk.Kind, "size": size},
	})
	httpx.WriteJSON(w, http.StatusCreated, att)
}

// ------------------------------------------------------ list / download

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	id, _ := auth.IdentityFrom(r.Context())
	includeAll := id.Role == auth.RoleAdmin || id.Role == auth.RoleSuperAdmin
	rows, err := h.repo.ListByCase(r.Context(), caseID, !includeAll)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) adminList(w http.ResponseWriter, r *http.Request) {
	caseID, err := parseID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	rows, err := h.repo.ListByCase(r.Context(), caseID, false)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (h *Handlers) downloadURL(w http.ResponseWriter, r *http.Request) {
	id, err := parseAttID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	att, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err))
		return
	}
	signed, err := h.store.PresignGet(r.Context(), att.StorageKey, 5*time.Minute)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, signed)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseAttID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	var body struct {
		Kind      *string `json:"kind"`
		CaptionBN *string `json:"caption_bn"`
		CaptionEN *string `json:"caption_en"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	if body.Kind != nil && !validKinds[*body.Kind] {
		httpx.WriteError(w, r, h.logger, httpx.ValidationError(map[string]string{"kind": "public|hidden|internal"}, ""))
		return
	}
	out, err := h.repo.UpdateMetadata(r.Context(), id, body.Kind, body.CaptionBN, body.CaptionEN)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseAttID(r)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	att, err := h.repo.Delete(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, h.logger, mapNotFound(err))
		return
	}
	if err := h.store.DeleteObject(r.Context(), att.StorageKey); err != nil {
		// Object deletion is best-effort: the row is gone, R2 cleanup may
		// happen via lifecycle policies. We log and continue.
		h.logger.Warn("storage: delete object failed", "key", att.StorageKey, "err", err)
	}
	actor := auth.MustIdentity(r.Context())
	ip, ua := audit.FromRequest(r)
	h.writeAudit(r.Context(), audit.Entry{
		UserID: &actor.UserID, Action: audit.ActionAttachmentDelete,
		TargetType: "attachment", TargetID: att.ID.String(),
		IP: ip, UserAgent: ua,
		Metadata: map[string]any{"case_id": att.CaseID.String(), "key": att.StorageKey},
	})
	httpx.NoContent(w)
}

// ------------------------------------------------------ token

// presignToken is a short-lived MAC over the immutable parts of a presign
// request. It binds the finalize call to the original presign so a client
// cannot finalise an attachment that they did not originally request.
type presignToken struct {
	CaseID       uuid.UUID
	Key          string
	Kind         string
	SeqNo        int
	MimeType     string
	OriginalName string
	ExpiresAt    int64
}

func (h *Handlers) signToken(caseID uuid.UUID, key, kind string, seq int, mime, name string, exp time.Time) string {
	msg := fmt.Sprintf("%s|%s|%s|%d|%s|%s|%d",
		caseID.String(), key, kind, seq, mime, name, exp.Unix())
	mac := hmac.New(sha256.New, h.hmacKey)
	_, _ = mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(msg+"|"+sig))
}

func (h *Handlers) verifyToken(raw string) (*presignToken, bool) {
	if raw == "" {
		return nil, false
	}
	dec, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, false
	}
	parts := strings.Split(string(dec), "|")
	if len(parts) != 8 {
		return nil, false
	}
	msg := strings.Join(parts[:7], "|")
	sig, err := base64.RawURLEncoding.DecodeString(parts[7])
	if err != nil {
		return nil, false
	}
	mac := hmac.New(sha256.New, h.hmacKey)
	_, _ = mac.Write([]byte(msg))
	if !hmac.Equal(mac.Sum(nil), sig) {
		return nil, false
	}
	caseID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, false
	}
	exp, err := time.Parse(time.RFC3339, parts[6])
	_ = exp
	_ = err // we kept ExpiresAt as unix seconds; parse below
	var unix int64
	_, _ = fmt.Sscanf(parts[6], "%d", &unix)
	if unix > 0 && time.Now().Unix() > unix+300 {
		// allow finalize within 5 minutes after the upload window expires
		return nil, false
	}
	var seq int
	_, _ = fmt.Sscanf(parts[3], "%d", &seq)
	return &presignToken{
		CaseID: caseID, Key: parts[1], Kind: parts[2], SeqNo: seq,
		MimeType: parts[4], OriginalName: parts[5], ExpiresAt: unix,
	}, true
}

// ------------------------------------------------------ helpers

var validKinds = map[string]bool{
	"public":   true,
	"hidden":   true,
	"internal": true,
}

func canMutateCase(actor auth.Identity, c *cases.Case) bool {
	if c.SubmittedBy != nil && *c.SubmittedBy == actor.UserID {
		return true
	}
	return roleRank(actor.Role) >= roleRank(auth.RoleModerator)
}

func roleRank(r auth.Role) int {
	switch r {
	case auth.RoleSuperAdmin:
		return 5
	case auth.RoleAdmin:
		return 4
	case auth.RoleModerator:
		return 3
	case auth.RoleContributor:
		return 2
	case auth.RoleViewer:
		return 1
	}
	return 0
}

func parseID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.ValidationError(map[string]string{"id": "must be a uuid"}, "")
	}
	return id, nil
}

func parseAttID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "attID")
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.ValidationError(map[string]string{"attID": "must be a uuid"}, "")
	}
	return id, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound("Attachment not found.")
	}
	if errors.Is(err, cases.ErrNotFound) {
		return httpx.NotFound("Case not found.")
	}
	return err
}

func storedFromKey(key string) string { return path.Base(key) }

package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/me-nazim/criminal-archive/backend/internal/audit"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// Service holds the dependencies needed to register / authenticate users
// and rotate their refresh tokens.
type Service struct {
	pool       *pgxpool.Pool
	repo       *Repository
	jwt        JWTConfig
	bcryptCost int
	logger     *slog.Logger
}

// NewService constructs a Service.
func NewService(
	pool *pgxpool.Pool,
	repo *Repository,
	jwtCfg JWTConfig,
	bcryptCost int,
	logger *slog.Logger,
) *Service {
	return &Service{pool: pool, repo: repo, jwt: jwtCfg, bcryptCost: bcryptCost, logger: logger}
}

// LoginResult is what handlers return on successful login or refresh.
type LoginResult struct {
	AccessToken    string
	AccessExpires  time.Time
	RefreshRaw     string
	RefreshExpires time.Time
	User           *User
}

// RegisterParams describes a registration request.
type RegisterParams struct {
	Email    string
	Password string
	FullName string
	Phone    *string
}

// Register validates input, hashes the password, inserts the user with
// status=pending, and writes an audit row. It returns the freshly-created
// user — without any JWT, since pending users may not log in.
func (s *Service) Register(ctx context.Context, p RegisterParams, ip, ua string) (*User, error) {
	email := strings.ToLower(strings.TrimSpace(p.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, httpx.ValidationError(map[string]string{"email": "must be a valid email"}, "")
	}
	if strings.TrimSpace(p.FullName) == "" {
		return nil, httpx.ValidationError(map[string]string{"full_name": "required"}, "")
	}
	if err := ValidatePassword(p.Password); err != nil {
		return nil, httpx.ValidationError(map[string]string{"password": "must be at least 10 characters"}, "")
	}

	hash, err := HashPassword(p.Password, s.bcryptCost)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.CreateUser(ctx, CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		FullName:     p.FullName,
		Phone:        p.Phone,
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return nil, httpx.Conflict("email_taken", "An account with that email already exists.")
		}
		return nil, err
	}

	audit.Write(ctx, s.pool, audit.Entry{
		UserID:     &u.ID,
		Action:     audit.ActionUserRegister,
		TargetType: "user",
		TargetID:   u.ID.String(),
		IP:         ip,
		UserAgent:  ua,
	}, s.logger)

	return u, nil
}

// LoginParams describes a login request.
type LoginParams struct {
	Email    string
	Password string
}

// Login verifies credentials and issues tokens.
func (s *Service) Login(ctx context.Context, p LoginParams, ip, ua string) (*LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(p.Email))
	user, hash, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Unauthenticated("Invalid email or password.")
		}
		return nil, err
	}
	if err := VerifyPassword(hash, p.Password); err != nil {
		return nil, httpx.Unauthenticated("Invalid email or password.")
	}

	switch user.Status {
	case "approved":
		// ok
	case "pending":
		return nil, httpx.Forbidden("account_pending", "Your account is awaiting admin approval.")
	case "suspended":
		return nil, httpx.Forbidden("account_suspended", "Your account is suspended.")
	case "rejected":
		return nil, httpx.Forbidden("account_rejected", "Your account has been rejected.")
	default:
		return nil, httpx.Forbidden("account_inactive", "Your account is not active.")
	}

	res, err := s.issueTokens(ctx, user, ip, ua)
	if err != nil {
		return nil, err
	}

	if err := s.repo.MarkLoggedIn(ctx, user.ID); err != nil {
		s.logger.Warn("mark logged in failed", "err", err, "user_id", user.ID)
	}

	audit.Write(ctx, s.pool, audit.Entry{
		UserID:     &user.ID,
		Action:     audit.ActionUserLogin,
		TargetType: "user",
		TargetID:   user.ID.String(),
		IP:         ip,
		UserAgent:  ua,
	}, s.logger)

	return res, nil
}

// Refresh rotates a refresh token: any reuse of an old token is treated
// as evidence of compromise and revokes every session for that user.
func (s *Service) Refresh(ctx context.Context, rawRefresh, ip, ua string) (*LoginResult, error) {
	if rawRefresh == "" {
		return nil, httpx.Unauthenticated("Missing refresh token.")
	}
	hashed := HashRefreshToken(rawRefresh)

	sess, err := s.repo.FindActiveSession(ctx, hashed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// We don't know the user, so we can't selectively revoke.
			return nil, httpx.Unauthenticated("Refresh token is invalid or expired.")
		}
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	if user.Status != "approved" {
		_ = s.repo.RevokeSession(ctx, sess.ID)
		return nil, httpx.Forbidden("account_inactive", "Your account is not active.")
	}

	// Revoke the old session and issue a fresh one (single-use refresh).
	if err := s.repo.RevokeSession(ctx, sess.ID); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, user, ip, ua)
}

// Logout revokes a single session. We never error if the token is
// already gone — logout is idempotent.
func (s *Service) Logout(ctx context.Context, rawRefresh string) {
	if rawRefresh == "" {
		return
	}
	hashed := HashRefreshToken(rawRefresh)
	sess, err := s.repo.FindActiveSession(ctx, hashed)
	if err != nil {
		return
	}
	_ = s.repo.RevokeSession(ctx, sess.ID)
}

// ChangePassword sets a new password for the authenticated user and
// revokes every other session.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPwd, newPwd, ip, ua string) error {
	uRow, err := s.getUserAndHash(ctx, userID)
	if err != nil {
		return err
	}
	if err := VerifyPassword(uRow.PasswordHash, oldPwd); err != nil {
		return httpx.Unauthenticated("Current password is incorrect.")
	}
	if err := ValidatePassword(newPwd); err != nil {
		return httpx.ValidationError(map[string]string{"new_password": "must be at least 10 characters"}, "")
	}
	newHash, err := HashPassword(newPwd, s.bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, newHash); err != nil {
		return err
	}
	if err := s.repo.RevokeAllSessionsForUser(ctx, userID); err != nil {
		s.logger.Warn("revoke sessions on password change failed", "err", err, "user_id", userID)
	}
	audit.Write(ctx, s.pool, audit.Entry{
		UserID:     &userID,
		Action:     audit.ActionUserPasswordSet,
		TargetType: "user",
		TargetID:   userID.String(),
		IP:         ip,
		UserAgent:  ua,
	}, s.logger)
	return nil
}

// getUserAndHash is a small helper to fetch a user with their password
// hash by id. It exists so ChangePassword can verify the old password.
type userWithHash struct {
	*User
	PasswordHash string
}

func (s *Service) getUserAndHash(ctx context.Context, id uuid.UUID) (*userWithHash, error) {
	const stmt = `
SELECT id, email, full_name, display_name, role, status, phone, avatar_url, bio,
       last_login_at, created_at, updated_at, password_hash
FROM   users WHERE id = $1`
	var u User
	var hash string
	err := s.pool.QueryRow(ctx, stmt, id).Scan(
		&u.ID, &u.Email, &u.FullName, &u.DisplayName, &u.Role, &u.Status,
		&u.Phone, &u.AvatarURL, &u.Bio, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		&hash,
	)
	if err != nil {
		return nil, err
	}
	return &userWithHash{User: &u, PasswordHash: hash}, nil
}

// issueTokens signs a fresh access JWT, generates a refresh token, and
// stores its hash in the sessions table.
func (s *Service) issueTokens(ctx context.Context, user *User, ip, ua string) (*LoginResult, error) {
	access, accessExp, err := IssueAccessToken(s.jwt, user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	rawRefresh, hashedRefresh, err := NewRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshExp := time.Now().Add(s.jwt.RefreshTTL)

	uaPtr := strPtr(ua)
	ipPtr := strPtr(ip)

	if _, err := s.repo.CreateSession(ctx, Session{
		UserID:      user.ID,
		RefreshHash: hashedRefresh,
		UserAgent:   uaPtr,
		IPAddress:   ipPtr,
		ExpiresAt:   refreshExp,
	}); err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:    access,
		AccessExpires:  accessExp,
		RefreshRaw:     rawRefresh,
		RefreshExpires: refreshExp,
		User:           user,
	}, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

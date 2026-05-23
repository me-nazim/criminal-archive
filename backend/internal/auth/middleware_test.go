package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func handlerOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestAuthenticator_RejectsMissingHeader(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, time.Minute)
	mw := Authenticator(cfg)(handlerOK())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticator_AcceptsValidToken(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, time.Minute)
	tok, _, err := IssueAccessToken(cfg, uuid.New(), "contributor")
	if err != nil {
		t.Fatal(err)
	}
	mw := Authenticator(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentityFrom(r.Context())
		if !ok {
			t.Error("identity missing from context")
		}
		if id.Role != RoleContributor {
			t.Errorf("role = %q, want contributor", id.Role)
		}
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuthenticator_ExpiredHasSpecificCode(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, -time.Minute) // already expired
	tok, _, _ := IssueAccessToken(cfg, uuid.New(), "viewer")
	mw := Authenticator(cfg)(handlerOK())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if !contains(rr.Body.String(), "token_expired") {
		t.Errorf("expected body to mention token_expired; got %s", rr.Body.String())
	}
}

func TestRequireMinRole(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, time.Minute)
	cases := []struct {
		role     Role
		min      Role
		wantCode int
	}{
		{RoleViewer, RoleViewer, 200},
		{RoleViewer, RoleContributor, 403},
		{RoleContributor, RoleModerator, 403},
		{RoleModerator, RoleModerator, 200},
		{RoleAdmin, RoleAdmin, 200},
		{RoleAdmin, RoleSuperAdmin, 403},
		{RoleSuperAdmin, RoleAdmin, 200},
	}
	for _, c := range cases {
		c := c
		t.Run(string(c.role)+"_vs_"+string(c.min), func(t *testing.T) {
			tok, _, _ := IssueAccessToken(cfg, uuid.New(), string(c.role))
			final := Authenticator(cfg)(RequireMinRole(c.min)(handlerOK()))
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			final.ServeHTTP(rr, req)
			if rr.Code != c.wantCode {
				t.Errorf("got %d, want %d", rr.Code, c.wantCode)
			}
		})
	}
}

func TestRequireRole_SuperAdminBypass(t *testing.T) {
	t.Parallel()
	cfg := testJWTConfig(t, time.Minute)
	tok, _, _ := IssueAccessToken(cfg, uuid.New(), "super_admin")
	// Allow only "moderator" — super_admin should still pass.
	final := Authenticator(cfg)(RequireRole(RoleModerator)(handlerOK()))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	final.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("super_admin should bypass RequireRole, got %d", rr.Code)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

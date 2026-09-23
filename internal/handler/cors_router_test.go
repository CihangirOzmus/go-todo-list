package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-todo-list/internal/auth"
	"go-todo-list/internal/middleware"
	"go-todo-list/internal/models"
)

// roleParser maps a bearer token literally to a role, so tests can send
// "Bearer user" / "Bearer admin" without minting real JWTs.
type roleParser struct{}

func (roleParser) Parse(tok string) (auth.Claims, error) {
	switch tok {
	case "user", "power_user", "admin":
		return auth.Claims{UserID: 1, Role: models.Role(tok)}, nil
	}
	return auth.Claims{}, errors.New("bad token")
}

const allowedOrigin = "http://localhost:5173"

// corsRouter mirrors main.go: CORS wraps the whole mux.
func corsRouter() http.Handler {
	r := NewRouter(
		NewAuthHandler(stubAuthSvc{}),
		NewTodoHandler(nil),
		NewAdminHandler(nil),
		roleParser{},
	)
	return middleware.CORS([]string{allowedOrigin})(r)
}

// A browser preflight carries no Authorization header, so it must be
// answered by CORS before RequireAuth/RequireRole ever see it, on every
// restricted route.
func TestCORS_PreflightBypassesAuthOnRestrictedRoutes(t *testing.T) {
	h := corsRouter()
	for _, path := range []string{"/me", "/lists", "/admin/lists", "/admin/users"} {
		req := httptest.NewRequest("OPTIONS", path, nil)
		req.Header.Set("Origin", allowedOrigin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "authorization")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("%s preflight: got %d, want 204", path, rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != allowedOrigin {
			t.Errorf("%s preflight: missing Allow-Origin", path)
		}
	}
}

// CORS must not weaken auth: the real request still needs a valid token and
// the right role. The 401/403 must also carry CORS headers so the React app
// can read the JSON error instead of seeing an opaque network failure.
func TestCORS_RestrictedRoutesStillEnforceAuth(t *testing.T) {
	h := corsRouter()
	cases := []struct {
		name  string
		path  string
		token string // "" = no Authorization header
		want  int
	}{
		{"no token on /me", "/me", "", http.StatusUnauthorized},
		{"bad token on /lists", "/lists", "garbage", http.StatusUnauthorized},
		{"no token on /admin/lists", "/admin/lists", "", http.StatusUnauthorized},
		{"user on /admin/lists", "/admin/lists", "user", http.StatusForbidden},
		{"no token on /admin/users", "/admin/users", "", http.StatusUnauthorized},
		{"user on /admin/users", "/admin/users", "user", http.StatusForbidden},
		{"power_user on /admin/users", "/admin/users", "power_user", http.StatusForbidden},
	}
	for _, tc := range cases {
		req := httptest.NewRequest("GET", tc.path, nil)
		req.Header.Set("Origin", allowedOrigin)
		if tc.token != "" {
			req.Header.Set("Authorization", "Bearer "+tc.token)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, rr.Code, tc.want)
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
			t.Errorf("%s: error response lost CORS header, got %q", tc.name, got)
		}
	}
}

// A disallowed origin gets the same auth verdict but no CORS headers, so the
// browser never exposes the response to the page.
func TestCORS_DisallowedOriginGetsNoHeadersOnRestrictedRoute(t *testing.T) {
	h := corsRouter()
	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin should be empty for a disallowed origin, got %q", got)
	}
}

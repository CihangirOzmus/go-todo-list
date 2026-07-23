package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-todo-list/internal/auth"
	"go-todo-list/internal/models"
)

type fakeParser struct {
	claims auth.Claims
	err    error
}

func (f fakeParser) Parse(_ string) (auth.Claims, error) { return f.claims, f.err }

func TestRequireAuth_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	rr := httptest.NewRecorder()
	RequireAuth(fakeParser{})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("next should not be called")
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d", rr.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer bogus")
	rr := httptest.NewRecorder()
	RequireAuth(fakeParser{err: errors.New("bad")})(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			t.Fatal("next should not be called")
		}),
	).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d", rr.Code)
	}
}

func TestRequireAuth_ValidTokenSetsClaims(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer any")
	rr := httptest.NewRecorder()

	want := auth.Claims{UserID: 7, Role: models.RolePowerUser}
	called := false
	RequireAuth(fakeParser{claims: want})(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			called = true
			got, ok := ClaimsFromContext(r.Context())
			if !ok {
				t.Fatal("no claims in ctx")
			}
			if got != want {
				t.Errorf("got %+v want %+v", got, want)
			}
		}),
	).ServeHTTP(rr, req)
	if !called {
		t.Fatal("next was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("code %d", rr.Code)
	}
}

func TestRequireRole(t *testing.T) {
	cases := []struct {
		name     string
		role     models.Role
		allowed  []models.Role
		wantCode int
	}{
		{"user rejected for admin-only", models.RoleUser, []models.Role{models.RoleAdmin}, http.StatusForbidden},
		{"admin allowed", models.RoleAdmin, []models.Role{models.RoleAdmin}, http.StatusOK},
		{"power_user in allowlist", models.RolePowerUser, []models.Role{models.RolePowerUser, models.RoleAdmin}, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer x")
			rr := httptest.NewRecorder()
			chain := RequireAuth(fakeParser{claims: auth.Claims{Role: tc.role}})(
				RequireRole(tc.allowed...)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})),
			)
			chain.ServeHTTP(rr, req)
			if rr.Code != tc.wantCode {
				t.Errorf("got %d want %d", rr.Code, tc.wantCode)
			}
		})
	}
}

func TestRequireRole_WithoutAuth(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	RequireRole(models.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d", rr.Code)
	}
}

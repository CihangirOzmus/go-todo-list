package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-todo-list/internal/models"
	"go-todo-list/internal/service"
)

type stubAdminSvc struct {
	err   error
	users []models.User

	gotID   int64
	gotRole models.Role
}

func (s *stubAdminSvc) ListUsers(_ context.Context) ([]models.User, error) {
	return s.users, s.err
}

func (s *stubAdminSvc) SetRole(_ context.Context, id int64, role models.Role) error {
	s.gotID, s.gotRole = id, role
	return s.err
}

func (s *stubAdminSvc) DeleteUser(_ context.Context, id int64) error {
	s.gotID = id
	return s.err
}

func TestAdminListUsers_NeverLeaksPasswordHash(t *testing.T) {
	svc := &stubAdminSvc{users: []models.User{
		{ID: 1, Username: "alice", Email: "a@x", PasswordHash: "$2a$topsecret", Role: models.RoleAdmin},
	}}
	rr := httptest.NewRecorder()
	NewAdminHandler(svc).ListUsers(rr, httptest.NewRequest("GET", "/admin/users", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "topsecret") {
		t.Errorf("password hash leaked: %s", rr.Body.String())
	}
	var out []models.User
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Username != "alice" {
		t.Errorf("got %+v", out)
	}
}

func TestAdminListUsers_EmptyIsArrayNotNull(t *testing.T) {
	rr := httptest.NewRecorder()
	NewAdminHandler(&stubAdminSvc{}).ListUsers(rr, httptest.NewRequest("GET", "/admin/users", nil))
	if got := strings.TrimSpace(rr.Body.String()); got != "[]" {
		t.Errorf("got %q, want []", got)
	}
}

func TestAdminListUsers_ServiceError(t *testing.T) {
	svc := &stubAdminSvc{err: errors.New("db down")}
	rr := httptest.NewRecorder()
	NewAdminHandler(svc).ListUsers(rr, httptest.NewRequest("GET", "/admin/users", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", rr.Code)
	}
}

func TestAdminSetRole(t *testing.T) {
	svc := &stubAdminSvc{}
	req := httptest.NewRequest("PUT", "/admin/users/3/role", strings.NewReader(`{"role":"power_user"}`))
	req.SetPathValue("id", "3")
	rr := httptest.NewRecorder()
	NewAdminHandler(svc).SetRole(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.gotID != 3 || svc.gotRole != models.RolePowerUser {
		t.Errorf("service saw id=%d role=%q", svc.gotID, svc.gotRole)
	}
}

func TestAdminSetRole_UnknownRoleIsRejected(t *testing.T) {
	// The service validates the role; the handler must map that to 400.
	svc := &stubAdminSvc{err: service.ErrValidation}
	req := httptest.NewRequest("PUT", "/admin/users/3/role", strings.NewReader(`{"role":"wizard"}`))
	req.SetPathValue("id", "3")
	rr := httptest.NewRecorder()
	NewAdminHandler(svc).SetRole(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rr.Code)
	}
}

func TestAdminSetRole_BadIDAndBadJSON(t *testing.T) {
	svc := &stubAdminSvc{}

	req := httptest.NewRequest("PUT", "/admin/users/abc/role", strings.NewReader(`{"role":"user"}`))
	req.SetPathValue("id", "abc")
	rr := httptest.NewRecorder()
	NewAdminHandler(svc).SetRole(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("bad id: got %d", rr.Code)
	}

	req = httptest.NewRequest("PUT", "/admin/users/3/role", strings.NewReader(`nope`))
	req.SetPathValue("id", "3")
	rr = httptest.NewRecorder()
	NewAdminHandler(svc).SetRole(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("bad json: got %d", rr.Code)
	}
	if svc.gotID != 0 {
		t.Errorf("service reached with id %d despite bad input", svc.gotID)
	}
}

func TestAdminDeleteUser(t *testing.T) {
	svc := &stubAdminSvc{}
	req := httptest.NewRequest("DELETE", "/admin/users/6", nil)
	req.SetPathValue("id", "6")
	rr := httptest.NewRecorder()
	NewAdminHandler(svc).DeleteUser(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("got %d", rr.Code)
	}
	if svc.gotID != 6 {
		t.Errorf("service saw id %d, want 6", svc.gotID)
	}
}

func TestAdminDeleteUser_NotFound(t *testing.T) {
	svc := &stubAdminSvc{err: service.ErrNotFound}
	req := httptest.NewRequest("DELETE", "/admin/users/6", nil)
	req.SetPathValue("id", "6")
	rr := httptest.NewRecorder()
	NewAdminHandler(svc).DeleteUser(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}

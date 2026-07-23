package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-todo-list/internal/models"
	"go-todo-list/internal/service"
)

type stubAuthSvc struct{}

func (stubAuthSvc) Register(_ context.Context, username, email, password string) (*models.User, error) {
	if len(password) < 6 {
		return nil, service.ErrValidation
	}
	if username == "taken" {
		return nil, service.ErrUserExists
	}
	return &models.User{ID: 1, Username: username, Email: email, Role: models.RoleUser}, nil
}

func (stubAuthSvc) Login(_ context.Context, email, password string) (string, error) {
	if password == "correct" {
		return "jwt.abc", nil
	}
	return "", service.ErrInvalidCredentials
}

func (stubAuthSvc) Me(_ context.Context, id int64) (*models.User, error) {
	return &models.User{ID: id, Username: "me", Role: models.RoleUser}, nil
}

func TestRegisterHandler_Success(t *testing.T) {
	h := NewAuthHandler(stubAuthSvc{})
	body := bytes.NewBufferString(`{"username":"a","email":"a@x","password":"hunter22"}`)
	req := httptest.NewRequest("POST", "/register", body)
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	var out models.User
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Username != "a" {
		t.Errorf("got %+v", out)
	}
	if out.PasswordHash != "" {
		t.Errorf("password hash leaked in response: %q", out.PasswordHash)
	}
}

func TestRegisterHandler_Conflict(t *testing.T) {
	h := NewAuthHandler(stubAuthSvc{})
	req := httptest.NewRequest("POST", "/register",
		bytes.NewBufferString(`{"username":"taken","email":"t@x","password":"hunter22"}`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("got %d", rr.Code)
	}
}

func TestRegisterHandler_BadJSON(t *testing.T) {
	h := NewAuthHandler(stubAuthSvc{})
	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString(`not json`))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d", rr.Code)
	}
}

func TestLoginHandler(t *testing.T) {
	h := NewAuthHandler(stubAuthSvc{})

	req := httptest.NewRequest("POST", "/login",
		bytes.NewBufferString(`{"email":"a@x","password":"correct"}`))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	var body struct{ Token string }
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Token == "" {
		t.Error("expected non-empty token")
	}

	req = httptest.NewRequest("POST", "/login",
		bytes.NewBufferString(`{"email":"a@x","password":"nope"}`))
	rr = httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d", rr.Code)
	}
}

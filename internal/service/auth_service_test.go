package service

import (
	"context"
	"errors"
	"testing"

	"go-todo-list/internal/models"
)

func TestRegister_Success(t *testing.T) {
	ctx := context.Background()
	svc := NewAuthService(newFakeUserRepo(), &stubIssuer{token: "TOK"})
	u, err := svc.Register(ctx, "alice", "a@example.com", "hunter22")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 {
		t.Error("id should be assigned")
	}
	if u.Role != models.RoleUser {
		t.Errorf("default role should be user, got %v", u.Role)
	}
	if u.PasswordHash == "hunter22" {
		t.Error("password stored in plaintext")
	}
}

func TestRegister_Validation(t *testing.T) {
	ctx := context.Background()
	svc := NewAuthService(newFakeUserRepo(), &stubIssuer{})
	cases := []struct {
		user, email, pw string
	}{
		{"", "a@x", "hunter22"},
		{"a", "", "hunter22"},
		{"a", "a@x", "short"},
	}
	for _, c := range cases {
		if _, err := svc.Register(ctx, c.user, c.email, c.pw); !errors.Is(err, ErrValidation) {
			t.Errorf("expected validation error for %+v, got %v", c, err)
		}
	}
}

func TestRegister_Conflict(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, &stubIssuer{})
	if _, err := svc.Register(ctx, "alice", "a@x", "hunter22"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, "alice", "b@x", "hunter22"); !errors.Is(err, ErrUserExists) {
		t.Errorf("expected user exists, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, &stubIssuer{token: "SIGNED"})
	if _, err := svc.Register(ctx, "alice", "a@x", "hunter22"); err != nil {
		t.Fatal(err)
	}
	tok, err := svc.Login(ctx, "a@x", "hunter22")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "SIGNED" {
		t.Errorf("got token %q", tok)
	}
}

func TestLogin_BadCredentials(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, &stubIssuer{token: "T"})
	_, _ = svc.Register(ctx, "alice", "a@x", "hunter22")

	if _, err := svc.Login(ctx, "a@x", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password should map to invalid credentials, got %v", err)
	}
	if _, err := svc.Login(ctx, "ghost@x", "any"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown user should map to invalid credentials, got %v", err)
	}
}

func TestMe(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, &stubIssuer{})
	u, _ := svc.Register(ctx, "alice", "a@x", "hunter22")
	got, err := svc.Me(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice" {
		t.Errorf("got %+v", got)
	}
	if _, err := svc.Me(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not found, got %v", err)
	}
}

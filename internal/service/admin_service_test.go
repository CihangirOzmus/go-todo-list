package service

import (
	"context"
	"errors"
	"testing"

	"go-todo-list/internal/models"
)

func TestAdminSetRole(t *testing.T) {
	ctx := context.Background()
	r := newFakeUserRepo()
	_ = r.Create(ctx, &models.User{Username: "u", Email: "u@x", PasswordHash: "x", Role: models.RoleUser})
	svc := NewAdminService(r)

	if err := svc.SetRole(ctx, 1, models.RolePowerUser); err != nil {
		t.Fatal(err)
	}
	got, _ := r.GetByID(ctx, 1)
	if got.Role != models.RolePowerUser {
		t.Errorf("got role %v", got.Role)
	}

	if err := svc.SetRole(ctx, 1, models.Role("garbage")); !errors.Is(err, ErrValidation) {
		t.Errorf("expected validation, got %v", err)
	}
	if err := svc.SetRole(ctx, 999, models.RoleAdmin); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not found, got %v", err)
	}
}

func TestAdminDeleteUser(t *testing.T) {
	ctx := context.Background()
	r := newFakeUserRepo()
	_ = r.Create(ctx, &models.User{Username: "u", Email: "u@x", PasswordHash: "x", Role: models.RoleUser})
	svc := NewAdminService(r)

	if err := svc.DeleteUser(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteUser(ctx, 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not found on second delete, got %v", err)
	}
}

func TestAdminListUsers(t *testing.T) {
	ctx := context.Background()
	r := newFakeUserRepo()
	_ = r.Create(ctx, &models.User{Username: "a", Email: "a@x", PasswordHash: "x", Role: models.RoleUser})
	_ = r.Create(ctx, &models.User{Username: "b", Email: "b@x", PasswordHash: "x", Role: models.RoleAdmin})
	svc := NewAdminService(r)
	out, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Errorf("got %d users", len(out))
	}
}

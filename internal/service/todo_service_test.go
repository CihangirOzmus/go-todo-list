package service

import (
	"context"
	"errors"
	"testing"

	"go-todo-list/internal/models"
)

func TestCreateAndListOwn(t *testing.T) {
	ctx := context.Background()
	repo := newFakeTodoRepo()
	svc := NewTodoService(repo)

	l, err := svc.CreateList(ctx, 1, "Groceries")
	if err != nil {
		t.Fatal(err)
	}
	if l.ID == 0 {
		t.Fatal("id not assigned")
	}
	if l.Title != "Groceries" {
		t.Errorf("title %q", l.Title)
	}

	out, err := svc.ListsForUser(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d lists", len(out))
	}
	// Different user sees nothing.
	other, err := svc.ListsForUser(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Errorf("bleed-over: user 2 got %d", len(other))
	}
}

func TestCreateEmptyTitleValidation(t *testing.T) {
	ctx := context.Background()
	svc := NewTodoService(newFakeTodoRepo())
	_, err := svc.CreateList(ctx, 1, "   ")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected validation, got %v", err)
	}
}

func TestGetListReadPermissions(t *testing.T) {
	ctx := context.Background()
	repo := newFakeTodoRepo()
	svc := NewTodoService(repo)
	l, _ := svc.CreateList(ctx, 1, "alice-list")

	// Owner can read.
	if _, err := svc.GetList(ctx, l.ID, Caller{UserID: 1, Role: models.RoleUser}); err != nil {
		t.Errorf("owner read failed: %v", err)
	}
	// Different regular user cannot.
	if _, err := svc.GetList(ctx, l.ID, Caller{UserID: 2, Role: models.RoleUser}); !errors.Is(err, ErrForbidden) {
		t.Errorf("expected forbidden, got %v", err)
	}
	// power_user can read cross-user.
	if _, err := svc.GetList(ctx, l.ID, Caller{UserID: 99, Role: models.RolePowerUser}); err != nil {
		t.Errorf("power_user read failed: %v", err)
	}
	// admin can read cross-user.
	if _, err := svc.GetList(ctx, l.ID, Caller{UserID: 99, Role: models.RoleAdmin}); err != nil {
		t.Errorf("admin read failed: %v", err)
	}
}

func TestListWritePermissions(t *testing.T) {
	ctx := context.Background()
	repo := newFakeTodoRepo()
	svc := NewTodoService(repo)
	l, _ := svc.CreateList(ctx, 1, "alice-list")

	// power_user is READ-only cross-user: writes should be forbidden.
	if err := svc.UpdateList(ctx, l.ID, "renamed", Caller{UserID: 99, Role: models.RolePowerUser}); !errors.Is(err, ErrForbidden) {
		t.Errorf("power_user should not write another user's list, got %v", err)
	}
	// admin can write cross-user.
	if err := svc.UpdateList(ctx, l.ID, "admin-renamed", Caller{UserID: 99, Role: models.RoleAdmin}); err != nil {
		t.Errorf("admin write failed: %v", err)
	}
	// owner can delete.
	if err := svc.DeleteList(ctx, l.ID, Caller{UserID: 1, Role: models.RoleUser}); err != nil {
		t.Errorf("owner delete failed: %v", err)
	}
}

func TestAllListsRoleGate(t *testing.T) {
	ctx := context.Background()
	svc := NewTodoService(newFakeTodoRepo())

	if _, err := svc.AllLists(ctx, Caller{Role: models.RoleUser}); !errors.Is(err, ErrForbidden) {
		t.Errorf("user should not access AllLists, got %v", err)
	}
	if _, err := svc.AllLists(ctx, Caller{Role: models.RolePowerUser}); err != nil {
		t.Errorf("power_user AllLists failed: %v", err)
	}
	if _, err := svc.AllLists(ctx, Caller{Role: models.RoleAdmin}); err != nil {
		t.Errorf("admin AllLists failed: %v", err)
	}
}

func TestAddAndDeleteTodo(t *testing.T) {
	ctx := context.Background()
	repo := newFakeTodoRepo()
	svc := NewTodoService(repo)
	l, _ := svc.CreateList(ctx, 1, "Groceries")

	tt, err := svc.AddTodo(ctx, l.ID, "milk", Caller{UserID: 1, Role: models.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if tt.ID == 0 {
		t.Fatal("todo id not assigned")
	}

	// non-owner rejected
	if _, err := svc.AddTodo(ctx, l.ID, "steal", Caller{UserID: 2, Role: models.RoleUser}); !errors.Is(err, ErrForbidden) {
		t.Errorf("expected forbidden, got %v", err)
	}

	// power_user cannot write; admin can.
	if err := svc.UpdateTodo(ctx, tt.ID, "milk", true, Caller{UserID: 99, Role: models.RolePowerUser}); !errors.Is(err, ErrForbidden) {
		t.Errorf("power_user write should be forbidden, got %v", err)
	}
	if err := svc.UpdateTodo(ctx, tt.ID, "milk", true, Caller{UserID: 99, Role: models.RoleAdmin}); err != nil {
		t.Errorf("admin write failed: %v", err)
	}

	// owner can delete
	if err := svc.DeleteTodo(ctx, tt.ID, Caller{UserID: 1, Role: models.RoleUser}); err != nil {
		t.Errorf("owner delete failed: %v", err)
	}
	// after delete, gone
	if err := svc.DeleteTodo(ctx, tt.ID, Caller{UserID: 1, Role: models.RoleUser}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not found, got %v", err)
	}
}

func TestUpdateList_NotFound(t *testing.T) {
	ctx := context.Background()
	svc := NewTodoService(newFakeTodoRepo())
	err := svc.UpdateList(ctx, 999, "x", Caller{UserID: 1, Role: models.RoleAdmin})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected not found, got %v", err)
	}
}

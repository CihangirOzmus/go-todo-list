package service

import (
	"context"
	"errors"
	"testing"

	"go-todo-list/internal/models"
)

var errDB = errors.New("connection reset by peer")

// errTodoRepo wraps the in-memory fake and fails one named method, so tests can
// check that an unexpected repository error surfaces as itself instead of being
// flattened into ErrNotFound or ErrForbidden.
type errTodoRepo struct {
	*fakeTodoRepo
	failOn string
}

func (e *errTodoRepo) fails(name string) bool { return e.failOn == name }

func (e *errTodoRepo) GetList(ctx context.Context, id int64) (*models.TodoList, error) {
	if e.fails("GetList") {
		return nil, errDB
	}
	return e.fakeTodoRepo.GetList(ctx, id)
}

func (e *errTodoRepo) GetListWithTodos(ctx context.Context, id int64) (*models.TodoList, error) {
	if e.fails("GetListWithTodos") {
		return nil, errDB
	}
	return e.fakeTodoRepo.GetListWithTodos(ctx, id)
}

func (e *errTodoRepo) GetTodo(ctx context.Context, id int64) (*models.Todo, error) {
	if e.fails("GetTodo") {
		return nil, errDB
	}
	return e.fakeTodoRepo.GetTodo(ctx, id)
}

func (e *errTodoRepo) CreateList(ctx context.Context, l *models.TodoList) error {
	if e.fails("CreateList") {
		return errDB
	}
	return e.fakeTodoRepo.CreateList(ctx, l)
}

func (e *errTodoRepo) CreateTodo(ctx context.Context, t *models.Todo) error {
	if e.fails("CreateTodo") {
		return errDB
	}
	return e.fakeTodoRepo.CreateTodo(ctx, t)
}

func (e *errTodoRepo) UpdateList(ctx context.Context, id int64, title string) error {
	if e.fails("UpdateList") {
		return errDB
	}
	return e.fakeTodoRepo.UpdateList(ctx, id, title)
}

func (e *errTodoRepo) DeleteList(ctx context.Context, id int64) error {
	if e.fails("DeleteList") {
		return errDB
	}
	return e.fakeTodoRepo.DeleteList(ctx, id)
}

func (e *errTodoRepo) UpdateTodo(ctx context.Context, id int64, content string, completed bool) error {
	if e.fails("UpdateTodo") {
		return errDB
	}
	return e.fakeTodoRepo.UpdateTodo(ctx, id, content, completed)
}

func (e *errTodoRepo) DeleteTodo(ctx context.Context, id int64) error {
	if e.fails("DeleteTodo") {
		return errDB
	}
	return e.fakeTodoRepo.DeleteTodo(ctx, id)
}

func (e *errTodoRepo) ListsByUser(ctx context.Context, uid int64) ([]models.TodoList, error) {
	if e.fails("ListsByUser") {
		return nil, errDB
	}
	return e.fakeTodoRepo.ListsByUser(ctx, uid)
}

func (e *errTodoRepo) AllLists(ctx context.Context) ([]models.TodoList, error) {
	if e.fails("AllLists") {
		return nil, errDB
	}
	return e.fakeTodoRepo.AllLists(ctx)
}

// newSeeded returns a service over a repo holding list 1 (owned by user 1) with
// todo 1 inside it, plus the repo so a test can choose which call fails.
func newSeeded(t *testing.T, failOn string) (*TodoService, *errTodoRepo) {
	t.Helper()
	base := newFakeTodoRepo()
	ctx := context.Background()
	l := &models.TodoList{UserID: 1, Title: "seed"}
	if err := base.CreateList(ctx, l); err != nil {
		t.Fatal(err)
	}
	if err := base.CreateTodo(ctx, &models.Todo{ListID: l.ID, Content: "seed"}); err != nil {
		t.Fatal(err)
	}
	repo := &errTodoRepo{fakeTodoRepo: base, failOn: failOn}
	return NewTodoService(repo), repo
}

func TestRepoErrorsPropagate(t *testing.T) {
	ctx := context.Background()
	owner := Caller{UserID: 1, Role: models.RoleUser}

	cases := []struct {
		name   string
		failOn string
		call   func(s *TodoService) error
	}{
		{"CreateList", "CreateList", func(s *TodoService) error {
			_, err := s.CreateList(ctx, 1, "t")
			return err
		}},
		{"ListsForUser", "ListsByUser", func(s *TodoService) error {
			_, err := s.ListsForUser(ctx, 1)
			return err
		}},
		{"GetList lookup", "GetListWithTodos", func(s *TodoService) error {
			_, err := s.GetList(ctx, 1, owner)
			return err
		}},
		{"UpdateList lookup", "GetList", func(s *TodoService) error {
			return s.UpdateList(ctx, 1, "t", owner)
		}},
		{"UpdateList write", "UpdateList", func(s *TodoService) error {
			return s.UpdateList(ctx, 1, "t", owner)
		}},
		{"DeleteList lookup", "GetList", func(s *TodoService) error {
			return s.DeleteList(ctx, 1, owner)
		}},
		{"DeleteList write", "DeleteList", func(s *TodoService) error {
			return s.DeleteList(ctx, 1, owner)
		}},
		{"AddTodo lookup", "GetList", func(s *TodoService) error {
			_, err := s.AddTodo(ctx, 1, "c", owner)
			return err
		}},
		{"AddTodo write", "CreateTodo", func(s *TodoService) error {
			_, err := s.AddTodo(ctx, 1, "c", owner)
			return err
		}},
		{"UpdateTodo todo lookup", "GetTodo", func(s *TodoService) error {
			return s.UpdateTodo(ctx, 1, "c", true, owner)
		}},
		{"UpdateTodo list lookup", "GetList", func(s *TodoService) error {
			return s.UpdateTodo(ctx, 1, "c", true, owner)
		}},
		{"UpdateTodo write", "UpdateTodo", func(s *TodoService) error {
			return s.UpdateTodo(ctx, 1, "c", true, owner)
		}},
		{"DeleteTodo todo lookup", "GetTodo", func(s *TodoService) error {
			return s.DeleteTodo(ctx, 1, owner)
		}},
		{"DeleteTodo list lookup", "GetList", func(s *TodoService) error {
			return s.DeleteTodo(ctx, 1, owner)
		}},
		{"DeleteTodo write", "DeleteTodo", func(s *TodoService) error {
			return s.DeleteTodo(ctx, 1, owner)
		}},
		{"AllLists", "AllLists", func(s *TodoService) error {
			_, err := s.AllLists(ctx, Caller{UserID: 9, Role: models.RoleAdmin})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newSeeded(t, tc.failOn)
			err := tc.call(svc)

			if !errors.Is(err, errDB) {
				t.Fatalf("got %v, want the underlying repo error", err)
			}
			// A transport failure must not be reported as a missing or
			// forbidden resource — that would hide an outage behind a 404/403.
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrForbidden) {
				t.Errorf("repo error masked as %v", err)
			}
		})
	}
}

// Authorisation must be decided before any write is attempted.
func TestForbiddenWritesNeverReachRepo(t *testing.T) {
	ctx := context.Background()
	stranger := Caller{UserID: 2, Role: models.RolePowerUser}

	// Each write fails loudly if the repo is reached, so a 403 proves the
	// service stopped before touching storage.
	for _, tc := range []struct {
		name   string
		failOn string
		call   func(s *TodoService) error
	}{
		{"UpdateList", "UpdateList", func(s *TodoService) error {
			return s.UpdateList(ctx, 1, "t", stranger)
		}},
		{"DeleteList", "DeleteList", func(s *TodoService) error {
			return s.DeleteList(ctx, 1, stranger)
		}},
		{"AddTodo", "CreateTodo", func(s *TodoService) error {
			_, err := s.AddTodo(ctx, 1, "c", stranger)
			return err
		}},
		{"UpdateTodo", "UpdateTodo", func(s *TodoService) error {
			return s.UpdateTodo(ctx, 1, "c", true, stranger)
		}},
		{"DeleteTodo", "DeleteTodo", func(s *TodoService) error {
			return s.DeleteTodo(ctx, 1, stranger)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newSeeded(t, tc.failOn)
			if err := tc.call(svc); !errors.Is(err, ErrForbidden) {
				t.Errorf("got %v, want ErrForbidden", err)
			}
		})
	}
}

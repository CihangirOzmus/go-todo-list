package service

import (
	"context"

	"go-todo-list/internal/models"
	"go-todo-list/internal/repository"
)

// fakeUserRepo is an in-memory UserRepo used by service tests.
type fakeUserRepo struct {
	users map[int64]*models.User
	seq   int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[int64]*models.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u *models.User) error {
	for _, ex := range f.users {
		if ex.Username == u.Username || ex.Email == u.Email {
			return repository.ErrConflict
		}
	}
	f.seq++
	u.ID = f.seq
	cp := *u
	f.users[u.ID] = &cp
	return nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) GetByID(_ context.Context, id int64) (*models.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) List(_ context.Context) ([]models.User, error) {
	out := make([]models.User, 0, len(f.users))
	for _, u := range f.users {
		out = append(out, *u)
	}
	return out, nil
}

func (f *fakeUserRepo) UpdateRole(_ context.Context, id int64, r models.Role) error {
	u, ok := f.users[id]
	if !ok {
		return repository.ErrNotFound
	}
	u.Role = r
	return nil
}

func (f *fakeUserRepo) Delete(_ context.Context, id int64) error {
	if _, ok := f.users[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.users, id)
	return nil
}

// stubIssuer implements TokenIssuer.
type stubIssuer struct {
	token string
	err   error
}

func (s *stubIssuer) Issue(_ int64, _ models.Role) (string, error) {
	return s.token, s.err
}

// fakeTodoRepo is an in-memory TodoRepo used by todo service tests.
type fakeTodoRepo struct {
	lists   map[int64]*models.TodoList
	todos   map[int64]*models.Todo
	listSeq int64
	todoSeq int64
}

func newFakeTodoRepo() *fakeTodoRepo {
	return &fakeTodoRepo{
		lists: map[int64]*models.TodoList{},
		todos: map[int64]*models.Todo{},
	}
}

func (f *fakeTodoRepo) CreateList(_ context.Context, l *models.TodoList) error {
	f.listSeq++
	l.ID = f.listSeq
	cp := *l
	f.lists[l.ID] = &cp
	return nil
}

func (f *fakeTodoRepo) GetList(_ context.Context, id int64) (*models.TodoList, error) {
	l, ok := f.lists[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *l
	return &cp, nil
}

func (f *fakeTodoRepo) GetListWithTodos(ctx context.Context, id int64) (*models.TodoList, error) {
	l, err := f.GetList(ctx, id)
	if err != nil {
		return nil, err
	}
	todos, _ := f.ListTodos(ctx, id)
	l.Todos = todos
	return l, nil
}

func (f *fakeTodoRepo) ListsByUser(_ context.Context, uid int64) ([]models.TodoList, error) {
	var out []models.TodoList
	for _, l := range f.lists {
		if l.UserID == uid {
			out = append(out, *l)
		}
	}
	return out, nil
}

func (f *fakeTodoRepo) AllLists(_ context.Context) ([]models.TodoList, error) {
	out := make([]models.TodoList, 0, len(f.lists))
	for _, l := range f.lists {
		out = append(out, *l)
	}
	return out, nil
}

func (f *fakeTodoRepo) UpdateList(_ context.Context, id int64, title string) error {
	l, ok := f.lists[id]
	if !ok {
		return repository.ErrNotFound
	}
	l.Title = title
	return nil
}

func (f *fakeTodoRepo) DeleteList(_ context.Context, id int64) error {
	if _, ok := f.lists[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.lists, id)
	for tid, t := range f.todos {
		if t.ListID == id {
			delete(f.todos, tid)
		}
	}
	return nil
}

func (f *fakeTodoRepo) CreateTodo(_ context.Context, t *models.Todo) error {
	f.todoSeq++
	t.ID = f.todoSeq
	cp := *t
	f.todos[t.ID] = &cp
	return nil
}

func (f *fakeTodoRepo) GetTodo(_ context.Context, id int64) (*models.Todo, error) {
	t, ok := f.todos[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTodoRepo) ListTodos(_ context.Context, listID int64) ([]models.Todo, error) {
	var out []models.Todo
	for _, t := range f.todos {
		if t.ListID == listID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (f *fakeTodoRepo) UpdateTodo(_ context.Context, id int64, content string, completed bool) error {
	t, ok := f.todos[id]
	if !ok {
		return repository.ErrNotFound
	}
	t.Content = content
	t.Completed = completed
	return nil
}

func (f *fakeTodoRepo) DeleteTodo(_ context.Context, id int64) error {
	if _, ok := f.todos[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.todos, id)
	return nil
}

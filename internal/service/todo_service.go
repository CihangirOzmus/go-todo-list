package service

import (
	"context"
	"errors"
	"strings"

	"go-todo-list/internal/models"
	"go-todo-list/internal/repository"
)

type TodoRepo interface {
	CreateList(ctx context.Context, l *models.TodoList) error
	GetList(ctx context.Context, id int64) (*models.TodoList, error)
	GetListWithTodos(ctx context.Context, id int64) (*models.TodoList, error)
	ListsByUser(ctx context.Context, userID int64) ([]models.TodoList, error)
	AllLists(ctx context.Context) ([]models.TodoList, error)
	UpdateList(ctx context.Context, id int64, title string) error
	DeleteList(ctx context.Context, id int64) error

	CreateTodo(ctx context.Context, t *models.Todo) error
	GetTodo(ctx context.Context, id int64) (*models.Todo, error)
	ListTodos(ctx context.Context, listID int64) ([]models.Todo, error)
	UpdateTodo(ctx context.Context, id int64, content string, completed bool) error
	DeleteTodo(ctx context.Context, id int64) error
}

// Caller identifies who is calling — used for ownership + role checks.
type Caller struct {
	UserID int64
	Role   models.Role
}

type TodoService struct {
	repo TodoRepo
}

func NewTodoService(r TodoRepo) *TodoService {
	return &TodoService{repo: r}
}

func (s *TodoService) CreateList(ctx context.Context, userID int64, title string) (*models.TodoList, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrValidation
	}
	l := &models.TodoList{UserID: userID, Title: title}
	if err := s.repo.CreateList(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *TodoService) ListsForUser(ctx context.Context, userID int64) ([]models.TodoList, error) {
	return s.repo.ListsByUser(ctx, userID)
}

func (s *TodoService) GetList(ctx context.Context, listID int64, caller Caller) (*models.TodoList, error) {
	l, err := s.repo.GetListWithTodos(ctx, listID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !canRead(caller, l) {
		return nil, ErrForbidden
	}
	return l, nil
}

func (s *TodoService) UpdateList(ctx context.Context, listID int64, title string, caller Caller) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrValidation
	}
	l, err := s.repo.GetList(ctx, listID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !canWrite(caller, l) {
		return ErrForbidden
	}
	return s.repo.UpdateList(ctx, listID, title)
}

func (s *TodoService) DeleteList(ctx context.Context, listID int64, caller Caller) error {
	l, err := s.repo.GetList(ctx, listID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !canWrite(caller, l) {
		return ErrForbidden
	}
	return s.repo.DeleteList(ctx, listID)
}

func (s *TodoService) AddTodo(ctx context.Context, listID int64, content string, caller Caller) (*models.Todo, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrValidation
	}
	l, err := s.repo.GetList(ctx, listID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !canWrite(caller, l) {
		return nil, ErrForbidden
	}
	t := &models.Todo{ListID: listID, Content: content}
	if err := s.repo.CreateTodo(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TodoService) UpdateTodo(ctx context.Context, todoID int64, content string, completed bool, caller Caller) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return ErrValidation
	}
	t, err := s.repo.GetTodo(ctx, todoID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	l, err := s.repo.GetList(ctx, t.ListID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !canWrite(caller, l) {
		return ErrForbidden
	}
	return s.repo.UpdateTodo(ctx, todoID, content, completed)
}

func (s *TodoService) DeleteTodo(ctx context.Context, todoID int64, caller Caller) error {
	t, err := s.repo.GetTodo(ctx, todoID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	l, err := s.repo.GetList(ctx, t.ListID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !canWrite(caller, l) {
		return ErrForbidden
	}
	return s.repo.DeleteTodo(ctx, todoID)
}

// AllLists — cross-user; only power_user and admin.
func (s *TodoService) AllLists(ctx context.Context, caller Caller) ([]models.TodoList, error) {
	if caller.Role != models.RolePowerUser && caller.Role != models.RoleAdmin {
		return nil, ErrForbidden
	}
	return s.repo.AllLists(ctx)
}

func canRead(c Caller, l *models.TodoList) bool {
	if l.UserID == c.UserID {
		return true
	}
	return c.Role == models.RolePowerUser || c.Role == models.RoleAdmin
}

// power_user has read-only across users; only owner + admin may write.
func canWrite(c Caller, l *models.TodoList) bool {
	if l.UserID == c.UserID {
		return true
	}
	return c.Role == models.RoleAdmin
}

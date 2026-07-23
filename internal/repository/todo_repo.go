package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-todo-list/internal/models"
)

type TodoRepo struct {
	pool *pgxpool.Pool
}

func NewTodoRepo(pool *pgxpool.Pool) *TodoRepo {
	return &TodoRepo{pool: pool}
}

// --- Lists ---

func (r *TodoRepo) CreateList(ctx context.Context, l *models.TodoList) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO todo_lists (user_id, title) VALUES ($1, $2)
		 RETURNING id, created_at, updated_at`,
		l.UserID, l.Title,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

func (r *TodoRepo) GetList(ctx context.Context, id int64) (*models.TodoList, error) {
	var l models.TodoList
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, created_at, updated_at FROM todo_lists WHERE id = $1`, id,
	).Scan(&l.ID, &l.UserID, &l.Title, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *TodoRepo) GetListWithTodos(ctx context.Context, id int64) (*models.TodoList, error) {
	l, err := r.GetList(ctx, id)
	if err != nil {
		return nil, err
	}
	todos, err := r.ListTodos(ctx, id)
	if err != nil {
		return nil, err
	}
	l.Todos = todos
	return l, nil
}

func (r *TodoRepo) ListsByUser(ctx context.Context, userID int64) ([]models.TodoList, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, title, created_at, updated_at
		 FROM todo_lists WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLists(rows)
}

func (r *TodoRepo) AllLists(ctx context.Context) ([]models.TodoList, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, title, created_at, updated_at FROM todo_lists ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLists(rows)
}

func (r *TodoRepo) UpdateList(ctx context.Context, id int64, title string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE todo_lists SET title = $1, updated_at = now() WHERE id = $2`, title, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TodoRepo) DeleteList(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM todo_lists WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Todos (items inside a list) ---

func (r *TodoRepo) CreateTodo(ctx context.Context, t *models.Todo) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO todos (list_id, content, completed) VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		t.ListID, t.Content, t.Completed,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *TodoRepo) GetTodo(ctx context.Context, id int64) (*models.Todo, error) {
	var t models.Todo
	err := r.pool.QueryRow(ctx,
		`SELECT id, list_id, content, completed, created_at, updated_at
		 FROM todos WHERE id = $1`, id,
	).Scan(&t.ID, &t.ListID, &t.Content, &t.Completed, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TodoRepo) ListTodos(ctx context.Context, listID int64) ([]models.Todo, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, list_id, content, completed, created_at, updated_at
		 FROM todos WHERE list_id = $1 ORDER BY id`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Todo
	for rows.Next() {
		var t models.Todo
		if err := rows.Scan(&t.ID, &t.ListID, &t.Content, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TodoRepo) UpdateTodo(ctx context.Context, id int64, content string, completed bool) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE todos SET content = $1, completed = $2, updated_at = now() WHERE id = $3`,
		content, completed, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TodoRepo) DeleteTodo(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanLists(rows pgx.Rows) ([]models.TodoList, error) {
	var out []models.TodoList
	for rows.Next() {
		var l models.TodoList
		if err := rows.Scan(&l.ID, &l.UserID, &l.Title, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-todo-list/internal/models"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *models.User) error {
	var role string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, role, created_at, updated_at`,
		u.Username, u.Email, u.PasswordHash, string(u.Role),
	).Scan(&u.ID, &role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	}
	u.Role = models.Role(role)
	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.scanOne(ctx,
		`SELECT id, username, email, password_hash, role, created_at, updated_at
		 FROM users WHERE email = $1`, email)
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return r.scanOne(ctx,
		`SELECT id, username, email, password_hash, role, created_at, updated_at
		 FROM users WHERE id = $1`, id)
}

func (r *UserRepo) scanOne(ctx context.Context, sql string, args ...any) (*models.User, error) {
	var u models.User
	var role string
	err := r.pool.QueryRow(ctx, sql, args...).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &role, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role = models.Role(role)
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, username, email, password_hash, role, created_at, updated_at
		 FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.User
	for rows.Next() {
		var u models.User
		var role string
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.Role = models.Role(role)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepo) UpdateRole(ctx context.Context, id int64, role models.Role) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE users SET role = $1, updated_at = now() WHERE id = $2`,
		string(role), id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

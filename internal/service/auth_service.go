package service

import (
	"context"
	"errors"
	"strings"

	"go-todo-list/internal/auth"
	"go-todo-list/internal/models"
	"go-todo-list/internal/repository"
)

type UserRepo interface {
	Create(ctx context.Context, u *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	List(ctx context.Context) ([]models.User, error)
	UpdateRole(ctx context.Context, id int64, role models.Role) error
	Delete(ctx context.Context, id int64) error
}

type TokenIssuer interface {
	Issue(userID int64, role models.Role) (string, error)
}

type AuthService struct {
	users  UserRepo
	tokens TokenIssuer
}

func NewAuthService(u UserRepo, t TokenIssuer) *AuthService {
	return &AuthService{users: u, tokens: t}
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" || email == "" || len(password) < 6 {
		return nil, ErrValidation
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         models.RoleUser,
	}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.users.GetByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, repository.ErrNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if err := auth.CheckPassword(u.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}
	return s.tokens.Issue(u.ID, u.Role)
}

func (s *AuthService) Me(ctx context.Context, id int64) (*models.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	return u, err
}

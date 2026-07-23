package service

import (
	"context"
	"errors"

	"go-todo-list/internal/models"
	"go-todo-list/internal/repository"
)

type AdminService struct {
	users UserRepo
}

func NewAdminService(u UserRepo) *AdminService {
	return &AdminService{users: u}
}

func (s *AdminService) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.users.List(ctx)
}

func (s *AdminService) SetRole(ctx context.Context, id int64, role models.Role) error {
	if !role.Valid() {
		return ErrValidation
	}
	err := s.users.UpdateRole(ctx, id, role)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *AdminService) DeleteUser(ctx context.Context, id int64) error {
	err := s.users.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

package service

import "errors"

var (
	ErrValidation         = errors.New("service: validation failed")
	ErrInvalidCredentials = errors.New("service: invalid credentials")
	ErrUserExists         = errors.New("service: user already exists")
	ErrNotFound           = errors.New("service: not found")
	ErrForbidden          = errors.New("service: forbidden")
)

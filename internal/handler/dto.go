package handler

import "go-todo-list/internal/models"

// RegisterRequest is the body of POST /register.
type RegisterRequest struct {
	Username string `json:"username" example:"alice"`
	Email    string `json:"email" example:"alice@example.com"`
	Password string `json:"password" example:"hunter22"`
}

// LoginRequest is the body of POST /login.
type LoginRequest struct {
	Email    string `json:"email" example:"alice@example.com"`
	Password string `json:"password" example:"hunter22"`
}

// LoginResponse is returned from POST /login.
type LoginResponse struct {
	Token string `json:"token"`
}

// CreateListRequest is the body of POST /lists.
type CreateListRequest struct {
	Title string `json:"title" example:"Groceries"`
}

// UpdateListRequest is the body of PUT /lists/{id}.
type UpdateListRequest struct {
	Title string `json:"title" example:"Renamed list"`
}

// CreateTodoRequest is the body of POST /lists/{id}/todos.
type CreateTodoRequest struct {
	Content string `json:"content" example:"buy milk"`
}

// UpdateTodoRequest is the body of PUT /todos/{id}.
type UpdateTodoRequest struct {
	Content   string `json:"content" example:"buy 2% milk"`
	Completed bool   `json:"completed"`
}

// SetRoleRequest is the body of PUT /admin/users/{id}/role.
type SetRoleRequest struct {
	Role models.Role `json:"role" example:"power_user"`
}

// StatusResponse is a small ack payload.
type StatusResponse struct {
	Status string `json:"status" example:"ok"`
}

// ErrorResponse is the shape returned on any error.
type ErrorResponse struct {
	Error string `json:"error" example:"validation failed"`
}

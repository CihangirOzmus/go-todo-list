package handler

import (
	"context"
	"net/http"

	"go-todo-list/internal/models"
)

type AdminSvc interface {
	ListUsers(ctx context.Context) ([]models.User, error)
	SetRole(ctx context.Context, id int64, role models.Role) error
	DeleteUser(ctx context.Context, id int64) error
}

type AdminHandler struct {
	svc AdminSvc
}

func NewAdminHandler(s AdminSvc) *AdminHandler {
	return &AdminHandler{svc: s}
}

// ListUsers returns every registered user (admin only).
//
//	@Summary	List all users
//	@Tags		admin
//	@Produce	json
//	@Success	200	{array}		models.User
//	@Failure	401	{object}	ErrorResponse
//	@Failure	403	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/admin/users [get]
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListUsers(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	if out == nil {
		out = []models.User{}
	}
	writeJSON(w, http.StatusOK, out)
}

// SetRole promotes or demotes a user (admin only).
//
//	@Summary	Change a user's role
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int				true	"User ID"
//	@Param		body	body		SetRoleRequest	true	"New role"
//	@Success	200		{object}	StatusResponse
//	@Failure	400		{object}	ErrorResponse
//	@Failure	401		{object}	ErrorResponse
//	@Failure	403		{object}	ErrorResponse
//	@Failure	404		{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/admin/users/{id}/role [put]
func (h *AdminHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var body SetRoleRequest
	if !decodeBody(w, r, &body) {
		return
	}
	if err := h.svc.SetRole(r.Context(), id, body.Role); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
}

// DeleteUser removes a user and their data (admin only).
//
//	@Summary		Delete a user
//	@Description	Cascades to the user's todo lists and todos.
//	@Tags			admin
//	@Param			id	path	int	true	"User ID"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteUser(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

package handler

import (
	"context"
	"net/http"

	"go-todo-list/internal/middleware"
	"go-todo-list/internal/models"
)

type AuthSvc interface {
	Register(ctx context.Context, username, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	Me(ctx context.Context, id int64) (*models.User, error)
}

type AuthHandler struct {
	svc AuthSvc
}

func NewAuthHandler(s AuthSvc) *AuthHandler {
	return &AuthHandler{svc: s}
}

// Register creates a new user account.
//
//	@Summary		Register
//	@Description	Create a new user. The role is always "user"; use the admin endpoints to promote.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest	true	"Registration payload"
//	@Success		201		{object}	models.User
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Router			/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body RegisterRequest
	if !decodeBody(w, r, &body) {
		return
	}
	u, err := h.svc.Register(r.Context(), body.Username, body.Email, body.Password)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// Login exchanges credentials for a JWT.
//
//	@Summary		Login
//	@Description	Exchange email and password for a JWT to send as `Authorization: Bearer <token>`.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Credentials"
//	@Success		200		{object}	LoginResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body LoginRequest
	if !decodeBody(w, r, &body) {
		return
	}
	tok, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{Token: tok})
}

// Me returns the authenticated user's profile.
//
//	@Summary	Current user
//	@Tags		auth
//	@Produce	json
//	@Success	200	{object}	models.User
//	@Failure	401	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	u, err := h.svc.Me(r.Context(), c.UserID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

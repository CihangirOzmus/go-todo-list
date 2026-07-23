package handler

import (
	"context"
	"net/http"

	"go-todo-list/internal/middleware"
	"go-todo-list/internal/models"
	"go-todo-list/internal/service"
)

type TodoSvc interface {
	CreateList(ctx context.Context, userID int64, title string) (*models.TodoList, error)
	ListsForUser(ctx context.Context, userID int64) ([]models.TodoList, error)
	GetList(ctx context.Context, id int64, caller service.Caller) (*models.TodoList, error)
	UpdateList(ctx context.Context, id int64, title string, caller service.Caller) error
	DeleteList(ctx context.Context, id int64, caller service.Caller) error

	AddTodo(ctx context.Context, listID int64, content string, caller service.Caller) (*models.Todo, error)
	UpdateTodo(ctx context.Context, id int64, content string, completed bool, caller service.Caller) error
	DeleteTodo(ctx context.Context, id int64, caller service.Caller) error

	AllLists(ctx context.Context, caller service.Caller) ([]models.TodoList, error)
}

type TodoHandler struct {
	svc TodoSvc
}

func NewTodoHandler(s TodoSvc) *TodoHandler {
	return &TodoHandler{svc: s}
}

// ListMine lists the caller's todo lists.
//
//	@Summary	List own todo lists
//	@Tags		lists
//	@Produce	json
//	@Success	200	{array}		models.TodoList
//	@Failure	401	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/lists [get]
func (h *TodoHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	c, _ := middleware.ClaimsFromContext(r.Context())
	out, err := h.svc.ListsForUser(r.Context(), c.UserID)
	if err != nil {
		handleErr(w, err)
		return
	}
	if out == nil {
		out = []models.TodoList{}
	}
	writeJSON(w, http.StatusOK, out)
}

// Create creates a new todo list owned by the caller.
//
//	@Summary	Create a todo list
//	@Tags		lists
//	@Accept		json
//	@Produce	json
//	@Param		body	body		CreateListRequest	true	"List payload"
//	@Success	201		{object}	models.TodoList
//	@Failure	400		{object}	ErrorResponse
//	@Failure	401		{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/lists [post]
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	c, _ := middleware.ClaimsFromContext(r.Context())
	var body CreateListRequest
	if !decodeBody(w, r, &body) {
		return
	}
	l, err := h.svc.CreateList(r.Context(), c.UserID, body.Title)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// Get returns a list with its todos.
//
//	@Summary		Get a todo list
//	@Description	Owners always allowed; power_user and admin can read any user's list.
//	@Tags			lists
//	@Produce		json
//	@Param			id	path		int	true	"List ID"
//	@Success		200	{object}	models.TodoList
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/lists/{id} [get]
func (h *TodoHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	l, err := h.svc.GetList(r.Context(), id, callerFrom(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// Update changes a list's title.
//
//	@Summary		Update a list's title
//	@Description	Owners and admin only. power_user is read-only across users.
//	@Tags			lists
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"List ID"
//	@Param			body	body		UpdateListRequest	true	"New title"
//	@Success		200		{object}	StatusResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/lists/{id} [put]
func (h *TodoHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var body UpdateListRequest
	if !decodeBody(w, r, &body) {
		return
	}
	if err := h.svc.UpdateList(r.Context(), id, body.Title, callerFrom(r)); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
}

// Delete removes a list and its todos.
//
//	@Summary		Delete a list
//	@Description	Owners and admin only. Cascades to the list's todos.
//	@Tags			lists
//	@Param			id	path	int	true	"List ID"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/lists/{id} [delete]
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteList(r.Context(), id, callerFrom(r)); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddTodo adds a todo item to a list.
//
//	@Summary	Add a todo to a list
//	@Tags		todos
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int					true	"List ID"
//	@Param		body	body		CreateTodoRequest	true	"Todo payload"
//	@Success	201		{object}	models.Todo
//	@Failure	400		{object}	ErrorResponse
//	@Failure	401		{object}	ErrorResponse
//	@Failure	403		{object}	ErrorResponse
//	@Failure	404		{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/lists/{id}/todos [post]
func (h *TodoHandler) AddTodo(w http.ResponseWriter, r *http.Request) {
	listID, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var body CreateTodoRequest
	if !decodeBody(w, r, &body) {
		return
	}
	t, err := h.svc.AddTodo(r.Context(), listID, body.Content, callerFrom(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// UpdateTodo modifies a todo's content and/or completed flag.
//
//	@Summary	Update a todo
//	@Tags		todos
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int					true	"Todo ID"
//	@Param		body	body		UpdateTodoRequest	true	"Todo payload"
//	@Success	200		{object}	StatusResponse
//	@Failure	400		{object}	ErrorResponse
//	@Failure	401		{object}	ErrorResponse
//	@Failure	403		{object}	ErrorResponse
//	@Failure	404		{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/todos/{id} [put]
func (h *TodoHandler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var body UpdateTodoRequest
	if !decodeBody(w, r, &body) {
		return
	}
	if err := h.svc.UpdateTodo(r.Context(), id, body.Content, body.Completed, callerFrom(r)); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
}

// DeleteTodo removes a single todo item.
//
//	@Summary	Delete a todo
//	@Tags		todos
//	@Param		id	path	int	true	"Todo ID"
//	@Success	204
//	@Failure	401	{object}	ErrorResponse
//	@Failure	403	{object}	ErrorResponse
//	@Failure	404	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/todos/{id} [delete]
func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteTodo(r.Context(), id, callerFrom(r)); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AllLists lists every user's todo lists (power_user + admin).
//
//	@Summary		Cross-user list feed
//	@Description	Available to power_user and admin only.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{array}		models.TodoList
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/lists [get]
func (h *TodoHandler) AllLists(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.AllLists(r.Context(), callerFrom(r))
	if err != nil {
		handleErr(w, err)
		return
	}
	if out == nil {
		out = []models.TodoList{}
	}
	writeJSON(w, http.StatusOK, out)
}

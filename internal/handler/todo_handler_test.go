package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-todo-list/internal/auth"
	"go-todo-list/internal/middleware"
	"go-todo-list/internal/models"
	"go-todo-list/internal/service"
)

// stubTodoSvc records what the handler passed down so the tests can assert the
// HTTP layer forwards the path id, the body and the caller unchanged.
type stubTodoSvc struct {
	err   error
	lists []models.TodoList

	gotUserID  int64
	gotID      int64
	gotTitle   string
	gotContent string
	gotDone    bool
	gotCaller  service.Caller
}

func (s *stubTodoSvc) CreateList(_ context.Context, userID int64, title string) (*models.TodoList, error) {
	s.gotUserID, s.gotTitle = userID, title
	if s.err != nil {
		return nil, s.err
	}
	return &models.TodoList{ID: 10, UserID: userID, Title: title}, nil
}

func (s *stubTodoSvc) ListsForUser(_ context.Context, userID int64) ([]models.TodoList, error) {
	s.gotUserID = userID
	return s.lists, s.err
}

func (s *stubTodoSvc) GetList(_ context.Context, id int64, c service.Caller) (*models.TodoList, error) {
	s.gotID, s.gotCaller = id, c
	if s.err != nil {
		return nil, s.err
	}
	return &models.TodoList{ID: id, UserID: 1, Title: "l"}, nil
}

func (s *stubTodoSvc) UpdateList(_ context.Context, id int64, title string, c service.Caller) error {
	s.gotID, s.gotTitle, s.gotCaller = id, title, c
	return s.err
}

func (s *stubTodoSvc) DeleteList(_ context.Context, id int64, c service.Caller) error {
	s.gotID, s.gotCaller = id, c
	return s.err
}

func (s *stubTodoSvc) AddTodo(_ context.Context, listID int64, content string, c service.Caller) (*models.Todo, error) {
	s.gotID, s.gotContent, s.gotCaller = listID, content, c
	if s.err != nil {
		return nil, s.err
	}
	return &models.Todo{ID: 5, ListID: listID, Content: content}, nil
}

func (s *stubTodoSvc) UpdateTodo(_ context.Context, id int64, content string, completed bool, c service.Caller) error {
	s.gotID, s.gotContent, s.gotDone, s.gotCaller = id, content, completed, c
	return s.err
}

func (s *stubTodoSvc) DeleteTodo(_ context.Context, id int64, c service.Caller) error {
	s.gotID, s.gotCaller = id, c
	return s.err
}

func (s *stubTodoSvc) AllLists(_ context.Context, c service.Caller) ([]models.TodoList, error) {
	s.gotCaller = c
	return s.lists, s.err
}

// authedRequest builds a request carrying claims the way RequireAuth would.
func authedRequest(method, target, body string, c service.Caller) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	ctx := middleware.WithClaims(r.Context(), auth.Claims{UserID: c.UserID, Role: c.Role})
	return r.WithContext(ctx)
}

var testCaller = service.Caller{UserID: 7, Role: models.RolePowerUser}

func TestListMine_UsesCallerIDAndNeverReturnsNull(t *testing.T) {
	svc := &stubTodoSvc{} // nil lists
	rr := httptest.NewRecorder()
	NewTodoHandler(svc).ListMine(rr, authedRequest("GET", "/lists", "", testCaller))

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.gotUserID != 7 {
		t.Errorf("service saw user %d, want 7", svc.gotUserID)
	}
	// A nil slice must serialise as [] so clients can iterate unconditionally.
	if got := strings.TrimSpace(rr.Body.String()); got != "[]" {
		t.Errorf("got %q, want []", got)
	}
}

func TestCreateList_AssignsCallerAsOwner(t *testing.T) {
	svc := &stubTodoSvc{}
	rr := httptest.NewRecorder()
	req := authedRequest("POST", "/lists", `{"title":"Groceries"}`, testCaller)
	NewTodoHandler(svc).Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.gotUserID != 7 || svc.gotTitle != "Groceries" {
		t.Errorf("service saw user=%d title=%q", svc.gotUserID, svc.gotTitle)
	}
	var out models.TodoList
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.ID != 10 {
		t.Errorf("got %+v", out)
	}
}

func TestCreateList_BadJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	req := authedRequest("POST", "/lists", `{`, testCaller)
	NewTodoHandler(&stubTodoSvc{}).Create(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d", rr.Code)
	}
}

func TestGetList_ForwardsIDAndCaller(t *testing.T) {
	svc := &stubTodoSvc{}
	req := authedRequest("GET", "/lists/42", "", testCaller)
	req.SetPathValue("id", "42")
	rr := httptest.NewRecorder()
	NewTodoHandler(svc).Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.gotID != 42 {
		t.Errorf("service saw id %d, want 42", svc.gotID)
	}
	// The caller is what the service authorises against, so it must survive.
	if svc.gotCaller != testCaller {
		t.Errorf("service saw caller %+v, want %+v", svc.gotCaller, testCaller)
	}
}

func TestUpdateList_OKAndForbidden(t *testing.T) {
	svc := &stubTodoSvc{}
	req := authedRequest("PUT", "/lists/3", `{"title":"Renamed"}`, testCaller)
	req.SetPathValue("id", "3")
	rr := httptest.NewRecorder()
	NewTodoHandler(svc).Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.gotID != 3 || svc.gotTitle != "Renamed" {
		t.Errorf("service saw id=%d title=%q", svc.gotID, svc.gotTitle)
	}

	// power_user may read but not write another user's list.
	svc = &stubTodoSvc{err: service.ErrForbidden}
	req = authedRequest("PUT", "/lists/3", `{"title":"Renamed"}`, testCaller)
	req.SetPathValue("id", "3")
	rr = httptest.NewRecorder()
	NewTodoHandler(svc).Update(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", rr.Code)
	}
}

func TestDeleteList_NoContentAndNotFound(t *testing.T) {
	svc := &stubTodoSvc{}
	req := authedRequest("DELETE", "/lists/9", "", testCaller)
	req.SetPathValue("id", "9")
	rr := httptest.NewRecorder()
	NewTodoHandler(svc).Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("got %d", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("204 must have an empty body, got %q", rr.Body.String())
	}

	svc = &stubTodoSvc{err: service.ErrNotFound}
	req = authedRequest("DELETE", "/lists/9", "", testCaller)
	req.SetPathValue("id", "9")
	rr = httptest.NewRecorder()
	NewTodoHandler(svc).Delete(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}

func TestAddTodo_UsesListIDFromPath(t *testing.T) {
	svc := &stubTodoSvc{}
	req := authedRequest("POST", "/lists/4/todos", `{"content":"buy milk"}`, testCaller)
	req.SetPathValue("id", "4")
	rr := httptest.NewRecorder()
	NewTodoHandler(svc).AddTodo(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.gotID != 4 || svc.gotContent != "buy milk" {
		t.Errorf("service saw listID=%d content=%q", svc.gotID, svc.gotContent)
	}
}

func TestUpdateTodo_ForwardsCompletedFlag(t *testing.T) {
	svc := &stubTodoSvc{}
	req := authedRequest("PUT", "/todos/8", `{"content":"done thing","completed":true}`, testCaller)
	req.SetPathValue("id", "8")
	rr := httptest.NewRecorder()
	NewTodoHandler(svc).UpdateTodo(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.gotID != 8 || svc.gotContent != "done thing" || !svc.gotDone {
		t.Errorf("service saw id=%d content=%q completed=%v", svc.gotID, svc.gotContent, svc.gotDone)
	}
}

func TestDeleteTodo(t *testing.T) {
	svc := &stubTodoSvc{}
	req := authedRequest("DELETE", "/todos/2", "", testCaller)
	req.SetPathValue("id", "2")
	rr := httptest.NewRecorder()
	NewTodoHandler(svc).DeleteTodo(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("got %d", rr.Code)
	}
	if svc.gotID != 2 {
		t.Errorf("service saw id %d, want 2", svc.gotID)
	}
}

func TestAllLists_ForbiddenForPlainUser(t *testing.T) {
	// The service is what rejects plain users; the handler must surface it as 403.
	svc := &stubTodoSvc{err: service.ErrForbidden}
	rr := httptest.NewRecorder()
	req := authedRequest("GET", "/admin/lists", "", service.Caller{UserID: 1, Role: models.RoleUser})
	NewTodoHandler(svc).AllLists(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rr.Code)
	}
}

func TestAllLists_EmptyIsArrayNotNull(t *testing.T) {
	rr := httptest.NewRecorder()
	req := authedRequest("GET", "/admin/lists", "", testCaller)
	NewTodoHandler(&stubTodoSvc{}).AllLists(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "[]" {
		t.Errorf("got %q, want []", got)
	}
}

// Every id-bearing route must reject a non-numeric path segment with 400 and
// never reach the service.
func TestTodoHandlers_InvalidIDParam(t *testing.T) {
	cases := []struct {
		name string
		call func(h *TodoHandler, w http.ResponseWriter, r *http.Request)
	}{
		{"Get", (*TodoHandler).Get},
		{"Update", (*TodoHandler).Update},
		{"Delete", (*TodoHandler).Delete},
		{"AddTodo", (*TodoHandler).AddTodo},
		{"UpdateTodo", (*TodoHandler).UpdateTodo},
		{"DeleteTodo", (*TodoHandler).DeleteTodo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubTodoSvc{}
			req := authedRequest("GET", "/x/abc", `{"title":"t","content":"c"}`, testCaller)
			req.SetPathValue("id", "abc")
			rr := httptest.NewRecorder()
			tc.call(NewTodoHandler(svc), rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("got %d, want 400", rr.Code)
			}
			if svc.gotID != 0 {
				t.Errorf("service was called with id %d", svc.gotID)
			}
		})
	}
}

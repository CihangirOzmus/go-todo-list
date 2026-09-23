package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-todo-list/internal/models"
	"go-todo-list/internal/service"
)

func TestHandleErr_MapsSentinelsToStatuses(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{service.ErrValidation, http.StatusBadRequest},
		{service.ErrInvalidCredentials, http.StatusUnauthorized},
		{service.ErrUserExists, http.StatusConflict},
		{service.ErrNotFound, http.StatusNotFound},
		{service.ErrForbidden, http.StatusForbidden},
		{errors.New("unexpected"), http.StatusInternalServerError},
		// Wrapped sentinels must still map to their own status.
		{fmt.Errorf("saving list: %w", service.ErrForbidden), http.StatusForbidden},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handleErr(rr, tc.err)
		if rr.Code != tc.want {
			t.Errorf("%v: got %d, want %d", tc.err, rr.Code, tc.want)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%v: content-type %q", tc.err, ct)
		}
		var body ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Errorf("%v: body is not JSON: %v", tc.err, err)
		}
		if body.Error == "" {
			t.Errorf("%v: empty error message", tc.err)
		}
	}
}

// An internal error must not echo the underlying failure to the client.
func TestHandleErr_DoesNotLeakInternalDetail(t *testing.T) {
	rr := httptest.NewRecorder()
	handleErr(rr, errors.New("pq: password authentication failed for user postgres"))
	if strings.Contains(rr.Body.String(), "postgres") {
		t.Errorf("internal detail leaked: %s", rr.Body.String())
	}
}

func TestWriteJSON_NilBodyWritesNothing(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusNoContent, nil)
	if rr.Code != http.StatusNoContent {
		t.Errorf("got %d", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", rr.Body.String())
	}
}

func TestIdParam(t *testing.T) {
	cases := []struct {
		raw    string
		want   int64
		wantOK bool
	}{
		{"42", 42, true},
		{"0", 0, true},
		{"-1", -1, true}, // parses; the service/DB simply finds no such row
		{"abc", 0, false},
		{"", 0, false},
		{"1.5", 0, false},
		{"9999999999999999999999", 0, false}, // overflows int64
	}
	for _, tc := range cases {
		req := httptest.NewRequest("GET", "/x", nil)
		req.SetPathValue("id", tc.raw)
		rr := httptest.NewRecorder()
		got, ok := idParam(rr, req, "id")

		if ok != tc.wantOK {
			t.Errorf("%q: ok=%v, want %v", tc.raw, ok, tc.wantOK)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("%q: got %d, want %d", tc.raw, got, tc.want)
		}
		if !ok && rr.Code != http.StatusBadRequest {
			t.Errorf("%q: got %d, want 400", tc.raw, rr.Code)
		}
	}
}

func TestCallerFrom(t *testing.T) {
	want := service.Caller{UserID: 7, Role: models.RolePowerUser}
	req := authedRequest("GET", "/x", "", want)
	if got := callerFrom(req); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// Without RequireAuth there are no claims, so the caller is the zero value —
// user 0 owns nothing, so it cannot be mistaken for a real user.
func TestCallerFrom_NoClaimsIsZeroValue(t *testing.T) {
	got := callerFrom(httptest.NewRequest("GET", "/x", nil))
	if got != (service.Caller{}) {
		t.Errorf("got %+v, want zero value", got)
	}
}

func TestDecodeBody_RejectsOversizedPayload(t *testing.T) {
	// Comfortably past maxBodyBytes.
	huge := `{"title":"` + strings.Repeat("a", 2*maxBodyBytes) + `"}`
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(huge))
	rr := httptest.NewRecorder()

	var body CreateListRequest
	if decodeBody(rr, req, &body) {
		t.Fatal("expected decodeBody to reject an oversized payload")
	}
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("got %d, want 413", rr.Code)
	}
}

func TestDecodeBody_AcceptsNormalPayload(t *testing.T) {
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(`{"title":"Groceries"}`))
	rr := httptest.NewRecorder()

	var body CreateListRequest
	if !decodeBody(rr, req, &body) {
		t.Fatalf("rejected a valid payload: %s", rr.Body.String())
	}
	if body.Title != "Groceries" {
		t.Errorf("got %q", body.Title)
	}
}

func TestMeHandler(t *testing.T) {
	h := NewAuthHandler(stubAuthSvc{})
	req := authedRequest("GET", "/me", "", service.Caller{UserID: 3, Role: models.RoleUser})
	rr := httptest.NewRecorder()
	h.Me(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	var out models.User
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.ID != 3 {
		t.Errorf("got id %d, want the caller's id 3", out.ID)
	}
}

func TestMeHandler_WithoutClaimsIsUnauthorized(t *testing.T) {
	h := NewAuthHandler(stubAuthSvc{})
	rr := httptest.NewRecorder()
	h.Me(rr, httptest.NewRequest("GET", "/me", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rr.Code)
	}
}

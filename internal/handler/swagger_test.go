package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "go-todo-list/docs"

	"go-todo-list/internal/auth"
	"go-todo-list/internal/middleware"
)

// nopParser satisfies middleware.Parser for router-only checks.
type nopParser struct{}

func (nopParser) Parse(_ string) (auth.Claims, error) { return auth.Claims{}, nil }

var _ middleware.Parser = nopParser{}

func TestSwaggerRoutes(t *testing.T) {
	r := NewRouter(
		NewAuthHandler(stubAuthSvc{}),
		NewTodoHandler(nil),
		NewAdminHandler(nil),
		nopParser{},
	)

	// doc.json should be valid JSON with the API title.
	req := httptest.NewRequest("GET", "/swagger/doc.json", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("doc.json: got %d body=%s", rr.Code, rr.Body.String())
	}
	var spec map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&spec); err != nil {
		t.Fatalf("doc.json is not valid JSON: %v", err)
	}
	info, _ := spec["info"].(map[string]any)
	if title, _ := info["title"].(string); title != "Todo List API" {
		t.Errorf("unexpected info.title: %v", info["title"])
	}

	// index.html should serve the Swagger UI.
	req = httptest.NewRequest("GET", "/swagger/index.html", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("index.html: got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "swagger") {
		t.Errorf("index.html body doesn't look like Swagger UI")
	}

	// Bare /swagger should redirect to /swagger/index.html.
	req = httptest.NewRequest("GET", "/swagger", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301 redirect from /swagger, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/swagger/index.html" {
		t.Errorf("redirect target: %q", loc)
	}
}

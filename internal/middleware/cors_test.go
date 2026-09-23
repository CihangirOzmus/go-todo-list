package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORS_AllowedOrigin(t *testing.T) {
	var called bool
	req := httptest.NewRequest("GET", "/lists", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173"})(okHandler(&called)).ServeHTTP(rr, req)

	if !called {
		t.Fatal("next should be called")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Allow-Origin = %q", got)
	}
	if got := rr.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	var called bool
	req := httptest.NewRequest("GET", "/lists", nil)
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173"})(okHandler(&called)).ServeHTTP(rr, req)

	if !called {
		t.Fatal("next should still be called; the browser enforces the block")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin should be empty, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	var called bool
	req := httptest.NewRequest("OPTIONS", "/lists", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	rr := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173"})(okHandler(&called)).ServeHTTP(rr, req)

	if called {
		t.Fatal("preflight must not reach next")
	}
	if rr.Code != http.StatusNoContent {
		t.Errorf("code = %d", rr.Code)
	}
	for _, h := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Max-Age",
	} {
		if rr.Header().Get(h) == "" {
			t.Errorf("missing %s", h)
		}
	}
}

func TestCORS_Wildcard(t *testing.T) {
	var called bool
	req := httptest.NewRequest("GET", "/lists", nil)
	req.Header.Set("Origin", "http://anything.example")
	rr := httptest.NewRecorder()

	CORS([]string{"*"})(okHandler(&called)).ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://anything.example" {
		t.Errorf("Allow-Origin = %q", got)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	var called bool
	req := httptest.NewRequest("GET", "/lists", nil)
	rr := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173"})(okHandler(&called)).ServeHTTP(rr, req)

	if !called {
		t.Fatal("same-origin / non-browser requests pass straight through")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin should be empty, got %q", got)
	}
}

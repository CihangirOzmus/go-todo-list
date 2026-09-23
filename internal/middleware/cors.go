package middleware

import (
	"net/http"
	"strings"
)

// CORS allows browser clients served from one of the given origins to call
// the API. Origins are compared exactly (scheme + host + port), so
// "http://localhost:5173" and "http://127.0.0.1:5173" are different entries.
// A single "*" entry allows any origin.
//
// Preflight (OPTIONS) requests are answered here and never reach the mux,
// so protected routes don't 401 on the preflight itself, which carries no
// Authorization header.
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(origins))
	any := false
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "*" {
			any = true
		} else if o != "" {
			allowed[o] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || !(any || allowed[origin]) {
				// Not a cross-origin browser request, or not an allowed origin.
				// Don't set CORS headers; the browser blocks the response.
				next.ServeHTTP(w, r)
				return
			}

			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			// The response varies per Origin, so caches must key on it.
			h.Add("Vary", "Origin")

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

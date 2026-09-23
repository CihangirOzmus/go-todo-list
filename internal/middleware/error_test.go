package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// writeErr builds its body with the JSON encoder, so a message containing
// quotes or backslashes cannot break out of the string and corrupt the payload.
func TestWriteErr_EscapesMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	writeErr(rr, http.StatusUnauthorized, `bad "token" \ here`)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type %q", ct)
	}
	var body struct{ Error string }
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("body is not valid JSON (%v): %s", err, rr.Body.String())
	}
	if body.Error != `bad "token" \ here` {
		t.Errorf("message round-tripped as %q", body.Error)
	}
}

// The middleware's own error shape must match the handler package's
// ErrorResponse, so clients see one error format everywhere.
func TestWriteErr_UsesErrorKey(t *testing.T) {
	rr := httptest.NewRecorder()
	writeErr(rr, http.StatusForbidden, "forbidden")

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["error"] != "forbidden" {
		t.Errorf("got %v, want a single \"error\" key", body)
	}
}

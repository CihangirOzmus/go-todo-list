package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-todo-list/internal/middleware"
	"go-todo-list/internal/service"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, ErrorResponse{Error: msg})
}

// maxBodyBytes caps request bodies so an unauthenticated caller cannot make us
// buffer an arbitrarily large payload.
const maxBodyBytes = 1 << 20 // 1 MiB

func decodeBody(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeErr(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeErr(w, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}

func idParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	v := r.PathValue(name)
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func callerFrom(r *http.Request) service.Caller {
	c, _ := middleware.ClaimsFromContext(r.Context())
	return service.Caller{UserID: c.UserID, Role: c.Role}
}

func handleErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		writeErr(w, http.StatusBadRequest, "validation failed")
	case errors.Is(err, service.ErrInvalidCredentials):
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, service.ErrUserExists):
		writeErr(w, http.StatusConflict, "user already exists")
	case errors.Is(err, service.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrForbidden):
		writeErr(w, http.StatusForbidden, "forbidden")
	default:
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}

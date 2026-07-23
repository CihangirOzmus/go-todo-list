package handler

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"go-todo-list/internal/middleware"
	"go-todo-list/internal/models"
)

func NewRouter(authH *AuthHandler, todoH *TodoHandler, adminH *AdminHandler, parser middleware.Parser) http.Handler {
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("POST /register", authH.Register)
	mux.HandleFunc("POST /login", authH.Login)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Swagger UI + spec. Redirect bare /swagger to the index for convenience.
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Middleware factories
	auth := middleware.RequireAuth(parser)
	powerOrAdmin := middleware.RequireRole(models.RolePowerUser, models.RoleAdmin)
	onlyAdmin := middleware.RequireRole(models.RoleAdmin)

	// Wrapping helpers
	protect := func(h http.HandlerFunc) http.Handler {
		return auth(http.HandlerFunc(h))
	}
	powerRole := func(h http.HandlerFunc) http.Handler {
		return auth(powerOrAdmin(http.HandlerFunc(h)))
	}
	adminRole := func(h http.HandlerFunc) http.Handler {
		return auth(onlyAdmin(http.HandlerFunc(h)))
	}

	// Authenticated (any role)
	mux.Handle("GET /me", protect(authH.Me))
	mux.Handle("GET /lists", protect(todoH.ListMine))
	mux.Handle("POST /lists", protect(todoH.Create))
	mux.Handle("GET /lists/{id}", protect(todoH.Get))
	mux.Handle("PUT /lists/{id}", protect(todoH.Update))
	mux.Handle("DELETE /lists/{id}", protect(todoH.Delete))
	mux.Handle("POST /lists/{id}/todos", protect(todoH.AddTodo))
	mux.Handle("PUT /todos/{id}", protect(todoH.UpdateTodo))
	mux.Handle("DELETE /todos/{id}", protect(todoH.DeleteTodo))

	// Cross-user read: power_user + admin
	mux.Handle("GET /admin/lists", powerRole(todoH.AllLists))

	// Admin only
	mux.Handle("GET /admin/users", adminRole(adminH.ListUsers))
	mux.Handle("PUT /admin/users/{id}/role", adminRole(adminH.SetRole))
	mux.Handle("DELETE /admin/users/{id}", adminRole(adminH.DeleteUser))

	return mux
}

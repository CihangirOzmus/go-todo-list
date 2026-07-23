package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	_ "go-todo-list/docs" // registers generated Swagger spec

	"go-todo-list/internal/auth"
	"go-todo-list/internal/config"
	"go-todo-list/internal/handler"
	"go-todo-list/internal/repository"
	"go-todo-list/internal/service"
)

// @title						Todo List API
// @version					1.0
// @description				Multi-user todo list REST API with JWT auth and role-based access.
// @description				Roles: `user` (own CRUD), `power_user` (own CRUD + read-all lists), `admin` (all + user management).
// @host						localhost:8080
// @BasePath					/
// @schemes					http https
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(rootCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	pingCtx, cancel := context.WithTimeout(rootCtx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	userRepo := repository.NewUserRepo(pool)
	todoRepo := repository.NewTodoRepo(pool)

	issuer := auth.NewIssuer(cfg.JWTSecret, cfg.JWTTTL)

	authSvc := service.NewAuthService(userRepo, issuer)
	todoSvc := service.NewTodoService(todoRepo)
	adminSvc := service.NewAdminService(userRepo)

	authH := handler.NewAuthHandler(authSvc)
	todoH := handler.NewTodoHandler(todoSvc)
	adminH := handler.NewAdminHandler(adminSvc)

	router := handler.NewRouter(authH, todoH, adminH, issuer)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           accessLog(router),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("go-todo-list listening on :%s (swagger at /swagger/index.html)", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-rootCtx.Done()
	log.Print("shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (s *statusWriter) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.code, time.Since(start))
	})
}

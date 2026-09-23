# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository context

`go-todo-list` is one of several sibling Go modules under `../` (e.g. `echo-command-line-args`, `go-slot-machine`, `concurrency-pipeline`) from a personal *The Go Programming Language* learning workspace. Unlike its siblings — small stdlib-only single-file exercises — this one is a full JSON REST API for a multi-user todo list with JWT auth, role-based access, and PostgreSQL persistence. The siblings do **not** share a workspace with it; each is an independent module.

## Common commands

Run from `go-todo-list/`:

```
go run .                              # build and run (needs env vars, see .env.example)
go build ./...                        # compile everything
go test ./...                         # all unit tests
go test -run TestName ./internal/...  # a single test
go test -race ./...                   # tests under the race detector
go vet ./...

docker compose up --build             # bring up Postgres + app on :8080
docker compose down -v                # drop everything including the pgdata volume
```

Required env vars (see `.env.example`): `DATABASE_URL`, `JWT_SECRET`, `JWT_TTL` (default `24h`), `PORT` (default `8080`).

### Regenerating Swagger docs

Swagger annotations live above each handler + on `main.go`. The generated `docs/` package (`docs.go`, `swagger.json`, `swagger.yaml`) is checked in, so `go build`/`go test` work without running the generator first.

After changing any `@Summary` / `@Param` / `@Router` / DTO struct — or the global `@title` block on `main.go` — regenerate:

```
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs --parseDependency --parseInternal
```

Pin the version to the `swaggo/swag` line in `go.mod`. Older binaries (e.g. a stale `swag` in `$GOPATH/bin`) fail to parse the Go 1.27 stdlib with `method must have no type parameters`.

Swagger UI is served at `http://localhost:8080/swagger/index.html` (bare `/swagger` redirects there). The raw spec is at `/swagger/doc.json`.

## Architecture

Layered, dependency-inverted:

```
handler → service → repository → pgxpool → Postgres
             ↑
           auth (bcrypt + JWT), models
```

- `internal/models` — `User`, `TodoList`, `Todo`, and the `Role` type (`user`, `power_user`, `admin`).
- `internal/auth` — `HashPassword`/`CheckPassword` (bcrypt) and `Issuer` (HS256 sign/parse). `Issuer` satisfies both `service.TokenIssuer` and `middleware.Parser`, so a single instance is threaded through the whole app.
- `internal/config` — env-var loader; fails fast on missing `DATABASE_URL` / `JWT_SECRET`.
- `internal/repository` — thin `pgxpool`-backed data access. `UserRepo` maps Postgres `23505` (unique_violation) to `ErrConflict`; both repos map `pgx.ErrNoRows` to `ErrNotFound`.
- `internal/service` — business logic. **Ownership + role checks live here**, not in middleware, because `power_user`/`admin` bypasses are per-operation. Services take repo *interfaces* (`UserRepo`, `TokenIssuer`, `TodoRepo`) so tests can plug in fakes (`internal/service/fakes_test.go`).
- `internal/middleware` — `RequireAuth` parses the `Bearer` header, stashes `auth.Claims` in `context.Context`. `RequireRole(...roles)` is a coarse gate for role-scoped routes (e.g., admin routes). Finer authorization is the service's job.
- `internal/handler` — HTTP boundary. `handler.go` has shared helpers (`writeJSON`, `handleErr` which maps `service.Err*` sentinels to HTTP statuses, `idParam`, `callerFrom`). `dto.go` defines the named request/response types Swagger references. `router.go` wires all routes with the stdlib `net/http` Go 1.22+ mux (`"POST /lists/{id}"` patterns) and applies middleware, plus mounts `/swagger/` via `swaggo/http-swagger`.
- `docs/` — **generated** by `swag` from annotations. Blank-imported by `main.go` so the `/swagger/doc.json` endpoint can find the spec. Do not edit by hand; re-run `swag init` (see below).

### Roles

- `user` — CRUD on own lists/todos.
- `power_user` — user + **read** every user's lists (via `GET /admin/lists` and cross-user `GET /lists/{id}`). Cannot write across users.
- `admin` — power_user + full write across users + `GET/PUT/DELETE /admin/users/...`.

Enforced by `canRead` / `canWrite` in `todo_service.go` and by `AdminService`.

### Auth flow

1. `POST /register` → bcrypt-hash, insert with role `user`.
2. `POST /login` → verify bcrypt hash, `Issuer.Issue` returns HS256 JWT with `{sub, role, iat, exp}`.
3. Protected routes: `RequireAuth(issuer)` parses `Authorization: Bearer <tok>` and puts claims in `ctx`. `service.Caller{UserID, Role}` is extracted via `handler.callerFrom(r)` and passed into services for ownership checks.

### Persistence

Schema in `migrations/001_init.sql`: `users`, `todo_lists`, `todos`, with FK cascades (delete a list → delete its todos; delete a user → delete their lists). Role is stored as `TEXT` with a `CHECK (role IN (...))` constraint (not a Postgres enum, to avoid pgx custom-type registration). Repos scan into a `string` and cast to `models.Role`.

Migrations are applied by mounting `./migrations` into the Postgres container's `/docker-entrypoint-initdb.d` (docker-compose) — they run **once**, on an empty data dir. To re-apply after schema changes: `docker compose down -v` then `up`.

## Testing

Only unit tests exist; there are no integration tests against Postgres. Repositories are therefore not directly tested — services are tested via in-memory fakes in `internal/service/fakes_test.go` (`fakeUserRepo`, `fakeTodoRepo`, `stubIssuer`). Handler tests use `net/http/httptest` with a stub service. Auth (bcrypt, JWT) and middleware are tested against their real implementations.

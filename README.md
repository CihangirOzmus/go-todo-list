# go-todo-list

A JSON REST API for a multi-user todo list. Users own multiple **lists**; each list holds any number of **todo** items. JWT-based auth with three roles:

| Role         | Own lists      | All users' lists    | User management     |
| ---          | ---            | ---                 | ---                 |
| `user`       | full CRUD      | —                   | —                   |
| `power_user` | full CRUD      | read-only           | —                   |
| `admin`      | full CRUD      | full CRUD           | list / role / delete |

Stack: Go 1.26 · stdlib `net/http` (Go 1.22+ mux) · PostgreSQL 18 · `pgx/v5` · `golang-jwt/v5` · `bcrypt` · Swagger via `swaggo/swag`.

---

## Functional requirements

- **FR-1 Registration.** Anyone can register an account with a username, email, and password. New accounts always get role `user`; there is no self-service role choice.
- **FR-2 Authentication.** `POST /login` exchanges email + password for a JWT (HS256). The token carries the caller's user id and role and expires after `JWT_TTL`. All protected endpoints reject requests without a valid `Authorization: Bearer <token>` header.
- **FR-3 Authorization / roles.** Three roles enforced per operation:
  - `user` — full CRUD on **their own** lists and todos only.
  - `power_user` — everything `user` can do, **plus read** any other user's lists and todos.
  - `admin` — everything `power_user` can do, **plus write** any user's lists/todos and manage users (list / change role / delete).
- **FR-4 Multiple lists per user.** A user can own any number of todo lists. A list has a title and belongs to exactly one user.
- **FR-5 Sub-todos inside a list.** Each list holds any number of todo items. A todo has a content string and a completed flag, and belongs to exactly one list.
- **FR-6 List CRUD.** Owners can create, read, rename, and delete their own lists. Admin can do the same across users.
- **FR-7 Todo CRUD.** Owners can add, update (content + completed), and delete todos in their lists. Admin can do the same across users.
- **FR-8 Cross-user read.** `power_user` and `admin` can `GET /lists/{id}` or `GET /admin/lists` for any user.
- **FR-9 User administration.** Admin can `GET /admin/users`, change a user's role via `PUT /admin/users/{id}/role`, and remove a user via `DELETE /admin/users/{id}`.
- **FR-10 Cascade on delete.** Deleting a list removes all its todos. Deleting a user removes all their lists and, transitively, their todos.
- **FR-11 API documentation.** An interactive OpenAPI (Swagger) UI is served alongside the API describing every endpoint, its request/response schema, and its auth requirements.

## Non-functional requirements

- **NFR-1 Security — passwords.** Passwords are stored as bcrypt hashes (`golang.org/x/crypto/bcrypt`), never in plaintext. The hash field is tagged `json:"-"` so it cannot leak in any response.
- **NFR-2 Security — tokens.** JWTs are signed with HS256 using a secret loaded from `JWT_SECRET`. Tokens carry `sub`, `role`, `iat`, and `exp`; the parser rejects wrong-signature and expired tokens.
- **NFR-3 Security — defense in depth.** Role gating happens at two layers: coarse `RequireRole` middleware on role-scoped routes (e.g. `/admin/*`) and fine-grained ownership checks inside services (`canRead` / `canWrite`). Neither is trusted alone.
- **NFR-4 Persistence.** State lives in PostgreSQL. Schema is versioned as SQL files under `migrations/` and applied via the Postgres image's `/docker-entrypoint-initdb.d`. Referential integrity (cascade delete on `user → lists → todos`) is enforced by foreign keys at the database, not by application code.
- **NFR-5 Reliability.** Startup fails fast on missing config or DB unreachable (`pool.Ping` on boot). The app shuts down gracefully on `SIGINT`/`SIGTERM` with a bounded drain window. Compose gates `app` on the DB's healthcheck so it never starts against an unready database.
- **NFR-6 Observability.** Every request emits an access-log line (`METHOD PATH STATUS DURATION`). `GET /healthz` is unauthenticated and returns 200 for liveness probes.
- **NFR-7 Portability / deployment.** The container is built from `golang:1.26-alpine` (builder) → `gcr.io/distroless/static-debian12:nonroot` (runtime). No CGO, no shell in the image, runs as the `nonroot` user. `docker compose up --build` is the only command needed to bring up the full stack.
- **NFR-8 Configurability.** All environment-specific values are env vars (`DATABASE_URL`, `JWT_SECRET`, `JWT_TTL`, `PORT`) — no code changes for a new environment, secret rotation, or token-lifetime tuning.
- **NFR-9 API contract.** JSON in / JSON out for every endpoint. Consistent status contract: `201` on create, `204` on delete, `200` on read/update, `400` bad payload, `401` missing/invalid token, `403` role or ownership rejects, `404` unknown id, `409` username/email conflict. Error payloads share the shape `{"error": "..."}`.
- **NFR-10 Robustness.** `http.Server.ReadHeaderTimeout` is set to 5s to blunt slow-header attacks. All handlers use request-scoped `context.Context` so client disconnects propagate to DB calls.
- **NFR-11 Testability.** Services depend on repository interfaces so unit tests use in-memory fakes — no test database required. `go test -race ./...` is clean.
- **NFR-12 Performance basics.** Connection pooling via `pgxpool`. Foreign key columns (`todo_lists.user_id`, `todos.list_id`) are indexed for owner-scoped and list-scoped queries.

---

## Prerequisites

- **Docker + Docker Compose** — required for the quickest path.
- **Go 1.26** — only if you want to run the app outside of Docker or run the test suite.

Nothing else. The app needs no local Postgres, no migration tool, no swag CLI at runtime.

---

## Quick start (Docker)

From the project root:

```
docker compose up --build
```

This brings up:
- **Postgres 18** on `localhost:5432` (user `todo`, password `todo`, database `todo`). Schema in `migrations/001_init.sql` is applied automatically on the first startup.
- **The API** on `localhost:8080`.

Stop everything (keep data):

```
docker compose down
```

Stop everything and wipe the database (so migrations run again on next `up`):

```
docker compose down -v
```

---

## Try the API

Health check:

```
curl -i http://localhost:8080/healthz          # → 200 OK
```

Register a user (default role is `user`):

```
curl -X POST http://localhost:8080/register \
  -H 'content-type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"hunter22"}'
```

Log in and capture the JWT:

```
TOKEN=$(curl -sS http://localhost:8080/login \
  -H 'content-type: application/json' \
  -d '{"email":"alice@example.com","password":"hunter22"}' | jq -r .token)
```

Create a list, add a todo, view it back:

```
curl -X POST http://localhost:8080/lists \
  -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"title":"Groceries"}'

curl -X POST http://localhost:8080/lists/1/todos \
  -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"content":"buy milk"}'

curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/lists/1
```

Unauthenticated call:

```
curl -i http://localhost:8080/lists    # → 401 missing bearer token
```

---

## Swagger UI

Open the interactive docs in a browser:

```
http://localhost:8080/swagger/index.html
```

- Click **Authorize** and paste `Bearer <your-token>` — every "Try it out" will attach it.
- Raw spec: `http://localhost:8080/swagger/doc.json` (OpenAPI 2.0) or `docs/swagger.yaml` in the repo.
- Protected routes are marked with 🔒.

---

## Promoting a user to `power_user` or `admin`

There is no self-service upgrade — only an existing `admin` can promote. Bootstrap the first admin by running SQL directly against the container:

```
docker compose exec db psql -U todo -d todo \
  -c "UPDATE users SET role='admin' WHERE username='alice';"
```

Then log in again to get a fresh JWT (the role is baked into the token):

```
TOKEN=$(curl -sS http://localhost:8080/login \
  -H 'content-type: application/json' \
  -d '{"email":"alice@example.com","password":"hunter22"}' | jq -r .token)

curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/admin/users
curl -X PUT http://localhost:8080/admin/users/2/role \
  -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"role":"power_user"}'
```

Valid role values: `user`, `power_user`, `admin`.

---

## Running outside of Docker

You need a Postgres reachable from your machine. If you're using the compose Postgres and just want to run the Go app locally against it, `docker compose up -d db` then in another terminal:

```
export DATABASE_URL='postgres://todo:todo@localhost:5432/todo?sslmode=disable'
export JWT_SECRET='local-dev-secret-please-change'
export JWT_TTL='24h'
export PORT='8080'

go run .
```

Or copy `.env.example` to `.env` and source it.

---

## Environment variables

| Variable       | Required | Default | Notes                                                  |
| ---            | ---      | ---     | ---                                                    |
| `DATABASE_URL` | yes      | —       | pgx-style URL, e.g. `postgres://user:pw@host:5432/db?sslmode=disable` |
| `JWT_SECRET`   | yes      | —       | HMAC-SHA256 signing key. Any non-empty string works; use something long and random in production. |
| `JWT_TTL`      | no       | `24h`   | Any `time.ParseDuration` value: `15m`, `24h`, `168h`, … |
| `PORT`         | no       | `8080`  | HTTP listen port                                       |

The app fails fast on startup if `DATABASE_URL` or `JWT_SECRET` is missing.

---

## Endpoint reference

Public:

| Method | Path        | Notes                                     |
| ---    | ---         | ---                                       |
| POST   | `/register` | Always creates a `user` role account      |
| POST   | `/login`    | Returns `{ "token": "..." }`              |
| GET    | `/healthz`  | Liveness check                            |
| GET    | `/swagger/…`| Swagger UI + spec (`/swagger` redirects)  |

Authenticated (any role) — send `Authorization: Bearer <token>`:

| Method | Path                    | Notes                              |
| ---    | ---                     | ---                                |
| GET    | `/me`                   | Current user profile               |
| GET    | `/lists`                | Caller's own lists                 |
| POST   | `/lists`                | Body: `{"title": "…"}`             |
| GET    | `/lists/{id}`           | With embedded `todos[]`; owner + power_user + admin |
| PUT    | `/lists/{id}`           | Body: `{"title": "…"}`; owner + admin |
| DELETE | `/lists/{id}`           | Owner + admin; cascades to todos   |
| POST   | `/lists/{id}/todos`     | Body: `{"content": "…"}`; owner + admin |
| PUT    | `/todos/{id}`           | Body: `{"content":"…","completed":true}` |
| DELETE | `/todos/{id}`           | Owner + admin                      |

`power_user` + `admin`:

| Method | Path            | Notes                              |
| ---    | ---             | ---                                |
| GET    | `/admin/lists`  | Every user's lists                 |

`admin` only:

| Method | Path                        | Notes                              |
| ---    | ---                         | ---                                |
| GET    | `/admin/users`              | List all users                     |
| PUT    | `/admin/users/{id}/role`    | Body: `{"role":"user\|power_user\|admin"}` |
| DELETE | `/admin/users/{id}`         | Cascades to lists + todos          |

HTTP status contract: `201` on create, `204` on delete, `200` on read/update, `400` bad payload, `401` missing/invalid token, `403` role or ownership rejects, `404` unknown id, `409` username/email already taken.

---

## Tests

```
go test ./...              # all unit tests
go test -race ./...        # with the race detector
go vet ./...
```

Unit tests do not require a running Postgres — services are tested against in-memory fake repositories.

---

## Regenerating Swagger docs

If you touch swag annotations, DTOs, or the `@title` block on `main.go`, regenerate `docs/`:

```
go install github.com/swaggo/swag/cmd/swag@latest    # once
$(go env GOPATH)/bin/swag init -g main.go -o docs --parseDependency --parseInternal
```

Then rebuild (or restart `docker compose up --build`). Never hand-edit files under `docs/` — they are regenerated wholesale.

# START.md — How to Run the FaceFinder Project

All commands are run from the project root unless stated otherwise:

```bash
cd "/run/media/sachin/Disk 2/facefinder"
```

---

## 1. Prerequisites

- Go 1.22+
- Node.js 18+ and npm
- Docker + Docker Compose

Check:

```bash
go version
node --version
docker --version
docker compose version
```

---

## 2. First-Time Setup

### 2.1 Create environment file

```bash
cp .env.example .env
```

For local development, the defaults in `.env.example` work as-is (fake face/storage/WhatsApp providers, no cloud keys needed).

For production, fill in the real values in `.env`:

```text
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
R2_ACCOUNT_ID=
R2_ACCESS_KEY_ID=
R2_SECRET_ACCESS_KEY=
WHATSAPP_GATEWAY_BASE_URL=
WHATSAPP_GATEWAY_API_KEY=
WHATSAPP_BUSINESS_NUMBER=
DATABASE_URL=            # Neon PostgreSQL connection string in production
```

### 2.2 Neon Auth (user login/signup)

Authentication uses **Neon Auth** (Better Auth hosted by Neon) when configured:

1. Neon Console → your project → **Auth** → Enable Auth → copy the **Auth URL**.
2. In `frontend/.env.local` (copy from `frontend/.env.local.example`) fill:

```text
NEON_AUTH_BASE_URL=https://ep-<xxx>.neonauth.<region>.aws.neon.tech/neondb/auth
NEON_AUTH_COOKIE_SECRET=<any random string, 32+ characters>
BETTER_AUTH_SECRET=<must exactly match BETTER_AUTH_SECRET in the project-root .env>
NEXT_PUBLIC_NEON_AUTH_ENABLED=true
```

3. Restart the frontend. Login/register then go through Neon Auth; the Next.js server trades the Neon session for a Go API token via `POST /api/v1/auth/exchange` (shared secret, server-to-server only). Photographer user/org data and authorization stay in the Go backend + Neon tables.
4. Offline fallback: leave `NEON_AUTH_BASE_URL` empty and `NEXT_PUBLIC_NEON_AUTH_ENABLED=false` — the built-in Go email/password auth is used instead.

### 2.3 Install frontend dependencies

```bash
cd frontend
npm install
cd ..
```

---

## 3. Start the Project (every time)

Open **4 terminals** and run one command in each, in this order.

### Terminal 1 — Infrastructure (hosted in production, Docker only for offline)

**Production (Neon DB + Upstash Redis):** nothing to start locally — both are hosted.
Just set these in the **project-root `.env`** (the Go API/worker/migrate load it automatically):

```text
NEON_DATABASE_URL=postgres://<user>:<password>@ep-<xxx>-pooler.<region>.aws.neon.tech/<db>?sslmode=require
UPSTASH_REDIS_URL=rediss://default:<password>@<your-db>.upstash.io:6379
```

- Neon: Console → Connect → enable **Connection pooling** (host contains `-pooler`) and copy the string as-is, keeping `?sslmode=require`.
- Upstash: Console → database → Connect → Redis URL. Use the TCP `rediss://` URL, **not** the REST URL/token — Asynq speaks the Redis protocol.

Config priority in the backend: `NEON_DATABASE_URL` → `DATABASE_URL` (offline), and `UPSTASH_REDIS_URL` → `REDIS_URL` (offline).
On startup the API, worker, and migrate tool log `database connected db=postgres://<host>/<db> neon=true` and `redis connected … upstash=true` (credentials never logged), or a loud `db unreachable` / `redis unreachable` error if a URL is wrong. First run against Neon also needs migrations (Terminal 2) — they create all tables, indexes, and RLS policies on Neon.

**Offline / local development (no Neon/Upstash):** leave `NEON_DATABASE_URL` and `UPSTASH_REDIS_URL` empty in `.env` and start the local containers instead:

```bash
cd "/run/media/sachin/Disk 2/facefinder"
docker compose up -d      # Postgres :5433 + local Redis :6379
docker compose ps
```

(Mixed mode also works — e.g. Neon DB + local Redis: set only `NEON_DATABASE_URL` and run `docker compose up -d redis`.)

### Terminal 2 — Database migrations (only needed once, or after new migrations)

```bash
cd "/run/media/sachin/Disk 2/facefinder/backend"
go run ./cmd/migrate
```

Expected output:

```text
applied migrations/001_init.sql
applied migrations/002_face_unique_per_event.sql
migrations up to date
```

### Terminal 3 — Go API server

```bash
cd "/run/media/sachin/Disk 2/facefinder/backend"
go run ./cmd/api
```

API runs at: **http://localhost:8080**

Health check (in any terminal):

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

### Terminal 4 — Go background worker (photo indexing + retention)

```bash
cd "/run/media/sachin/Disk 2/facefinder/backend"
go run ./cmd/worker
```

### Terminal 5 — Next.js frontend

Development mode (hot reload):

```bash
cd "/run/media/sachin/Disk 2/facefinder/frontend"
npm run dev
```

**OR** production mode:

```bash
cd "/run/media/sachin/Disk 2/facefinder/frontend"
npm run build
npm start
```

Frontend runs at: **http://localhost:3000**

---

## 4. Open the App

| What | URL |
|---|---|
| Landing page | http://localhost:3000 |
| Register | http://localhost:3000/register |
| Login | http://localhost:3000/login |
| Dashboard | http://localhost:3000/dashboard |
| Events | http://localhost:3000/events |
| Create event | http://localhost:3000/events/new |
| API health | http://localhost:8080/healthz |

### Existing local test account (if the local database was already seeded)

```text
Email:    test@example.com
Password: password123
```

Contains organization **"Test Studio"** and event **"Rahul Wedding"**.
Otherwise, register a fresh account at http://localhost:3000/register.

---

## 5. Quick End-to-End Check

1. Login → create an organization (first time only).
2. Create an event → copy the WhatsApp link / regenerate the QR from the event page (the event token is shown only at creation/regeneration — it is stored hashed).
3. Open the event → **Upload Photos** (batch upload supported; indexing happens in the background via the worker).
4. Watch processing stats: `Ready / No Face Found / Failed`, `Faces Indexed`.
5. Guest flow (needs a real WhatsApp gateway configured): guest scans the QR → sends the prefilled `START <token>` → sends one clear selfie → receives previews + a private gallery link at `http://localhost:3000/g/{token}`.

Without WhatsApp/AWS/R2 credentials the app uses local fake providers, so the full UI/API/worker flow still works — only real face matching and real WhatsApp delivery require the production keys in `.env`.

---

## 6. Build & Test Commands

### Backend

```bash
cd "/run/media/sachin/Disk 2/facefinder/backend"
go mod tidy
go vet ./...
go test ./...
go build -o /tmp/ff-api ./cmd/api
go build -o /tmp/ff-worker ./cmd/worker
go build -o /tmp/ff-migrate ./cmd/migrate
```

Run the built binaries instead of `go run` (optional):

```bash
/tmp/ff-migrate
/tmp/ff-api
/tmp/ff-worker
```

> Note: binaries are built from `backend/`, and `/tmp/ff-migrate` must also be **run from `backend/`** (it reads `migrations/*.sql` relative to the current directory).

### Frontend

```bash
cd "/run/media/sachin/Disk 2/facefinder/frontend"
npm install
npx tsc --noEmit
npm run build
```

---

## 7. Stop the Project

Stop API / worker / frontend: press `Ctrl+C` in each of their terminals.

Stop database + Redis:

```bash
cd "/run/media/sachin/Disk 2/facefinder"
docker compose down
```

Stop containers **and delete all local database data** (fresh start next time):

```bash
docker compose down -v
```

---

## 8. Troubleshooting

| Problem | Fix |
|---|---|
| `bind host port 0.0.0.0:5432 ... address already in use` | Expected on machines with a local PostgreSQL — this project maps Postgres to **5433**, not 5432. Make sure `docker-compose.yml` has `"5433:5432"` and `DATABASE_URL` in `.env` uses `localhost:5433`. |
| `go.mod file not found` | You are in the wrong folder — backend commands must run from `backend/`, frontend commands from `frontend/`. |
| API starts but uploads never finish processing | The **worker** (Terminal 4) is not running, or Redis is down — check `docker compose ps`. |
| Photos stuck in `Queued` | The **worker** (Terminal 4) is not running, or Redis is unreachable — check the worker log for `redis connected` / `redis unreachable`; if using Upstash, verify `UPSTASH_REDIS_URL` starts with `rediss://` in the project-root `.env`. Offline: check `docker compose ps` for Redis. |
| Frontend loads but API calls fail | API (Terminal 3) is not running, or `NEXT_PUBLIC_API_URL` is wrong — it defaults to `http://localhost:8080`. Check `curl http://localhost:8080/healthz`. |
| Port 8080 / 3000 already in use | Stop the old process: `pkill -x ff-api`, or find it with `lsof -i :8080` / `lsof -i :3000`. |
| Database connection refused | Offline: run `docker compose up -d`, wait a few seconds, then re-run migrations. Neon: check the API/migrate log for `db unreachable`, verify `NEON_DATABASE_URL` in the project-root `.env` uses the **-pooler** host and `sslmode=require`, and that your IP/network can reach Neon (first connection may take a few seconds while Neon's compute wakes from suspend). |
| Need a completely fresh database | `docker compose down -v && docker compose up -d`, then re-run migrations (Section 3, Terminal 2) and register a new account. |

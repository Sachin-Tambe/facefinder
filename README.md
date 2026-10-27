# Event Face Photo Finder + WhatsApp Delivery

Multi-tenant SaaS: photographers upload event photos → AWS Rekognition indexes every face into a **per-event** collection → guests scan an event QR → send a selfie on WhatsApp → get their photos via WhatsApp previews + a private, expiring gallery.

**This is face search / photo discovery — not verified biometric identity.**

## Locked stack
| Layer | Choice |
|---|---|
| Frontend | Next.js + TypeScript |
| Backend | Go (API + worker, single module) |
| DB | Neon PostgreSQL (metadata only — never photo bytes) |
| Auth | JWT (bcrypt) issued by Go API; seam for Better Auth / Neon Auth in `internal/middleware` + `frontend/src/lib/api.ts` |
| Storage | Cloudflare R2 (private, presigned PUT / signed GET) |
| Face AI | AWS Rekognition — one collection per event (`evt_<id>`) |
| Jobs | Redis + Asynq (retries, backoff, idempotency) |
| WhatsApp | Existing gateway via `WhatsAppProvider` interface |
| Images | `disintegration/imaging` (pure Go). Swap point for libvips (`govips`) is `internal/service/variants.go` — interface is one function. |

Provider interfaces (`FaceProvider`, `StorageProvider`, `WhatsAppProvider`) keep business logic decoupled — see `backend/internal/{face,storage,whatsapp}`.
Without cloud credentials the API boots with **fake providers** (deterministic, dev-only) so the whole flow is testable end-to-end. Set real env vars and they are replaced automatically.

## Quick start
```bash
cp .env.example .env          # fill DATABASE_URL etc. For Redis, set UPSTASH_REDIS_URL=rediss://... (Upstash console, TCP URL) — leave empty for offline local Redis
docker compose up -d          # postgres (+ local redis for offline dev; use `up -d postgres` only when on Upstash)
cd backend && go run ./cmd/migrate && go run ./cmd/api &   # :8080
cd backend && go run ./cmd/worker &                        # asynq worker + retention scheduler
cd frontend && npm install && npm run dev                  # :3000
```
Register → create organization → create event (creates Rekognition collection + QR) → upload photos (presigned → R2 → async indexing) → print QR → guest sends `START <token>` then a selfie on WhatsApp.

## Layout
```
backend/cmd/{api,worker,migrate}   frontend/src/app/{(auth),dashboard,events,g}
backend/internal/{config,database,model,store,service,httpapi,jobs,face,storage,whatsapp,middleware,logging}
backend/migrations/                docs/ARCHITECTURE.md · docs/AWS_IAM.md
```
## Retention (independent clocks, §31–34)
Selfie 60 min · WhatsApp session 24 h · Gallery 24 h · Face vectors 30 d (`DeleteCollection`, R2 untouched) · Originals 180 d (R2 delete, only if `auto_delete_enabled`).

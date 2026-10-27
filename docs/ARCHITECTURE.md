# Architecture & acceptance mapping

Flow: Photographer (Next.js) → Go API → Neon Postgres / R2 (presigned) → Redis+Asynq → AWS Rekognition (`evt_<id>` per event) · Guest: QR → WhatsApp gateway webhook → session (phone+gateway → event, TTL) → selfie → DetectFaces (0/>1 rejected) → SearchFacesByImage **in that event's collection only** → FaceId→photo_faces→photo (dedup by photo, best similarity, sorted) → ≤5 WhatsApp previews + private `/g/{token}` gallery (token hashed, expiring, revocable).

Isolation: every business query filters `organization_id`; membership checked in `requireOrg`/`Role`; Postgres RLS enabled as backstop. No global collection, no cross-event search — `ActiveSession` is the only path to a search, and no session ⇒ no search.

Thresholds are env-configured (`FACE_MATCH_THRESHOLD_STRONG/POSSIBLE`, `SEARCH_FACE_THRESHOLD`) — calibrate per §47 with real event photos (glasses, side profiles, low light, group shots); never hardcode.

Known seams / TODO before prod: (1) multipart R2 upload for >25MB files (presign currently single PUT, 25MB cap); (2) Redis-backed rate limiter + Sentry wiring (hooks in middleware/logging via slog `request_id/org/event/photo/search/job` fields); (3) Better Auth: replace JWT verifier in `middleware.Auth` — claims contract is `sub=user_id`; (4) plan/quota table (org.plan already present); (5) WhatsApp gateway path mapping in `whatsapp.Gateway` if the company gateway differs.

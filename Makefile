.PHONY: dev migrate api worker web test
dev: ; docker compose up -d
migrate: ; cd backend && go run ./cmd/migrate
api: ; cd backend && go run ./cmd/api
worker: ; cd backend && go run ./cmd/worker
web: ; cd frontend && npm run dev
test: ; cd backend && go vet ./... && go test ./... ; cd frontend && npx tsc --noEmit

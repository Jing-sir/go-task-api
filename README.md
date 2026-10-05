# go-task-api

Monorepo layout:

- `backend/` Go API service
- `web/` Vue 3 frontend

Run locally:

```bash
cd backend && go run ./cmd/server
cd web && pnpm install && pnpm dev
```

Build checks:

```bash
cd backend && go test ./...
cd web && pnpm build
```

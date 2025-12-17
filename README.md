# Worker Pool
A minimal Go HTTP API to manage tasks using an in-memory, concurrency-safe store.

# Implementation Phases
- API endpoints + clean layering (store/service/http) + graceful shutdown.

## Requirements
- Go 1.25.5+

## Project Structure (so far)
- `cmd/server/main.go` — Wiring + HTTP server + graceful shutdown
- `internal/config` — Runtime config (port, workers, pool size, shutdown timeout)
- `internal/domain` — Task
- `internal/store/memory` — In-memory task store (map + RWMutex, incremental int64 ID)
- `internal/service` — Use-cases + validation + error mapping
- `internal/http/handlers` — Endpoints
- `internal/router` — Routes using `net/http` patterns (Go 1.22+ style)

## Run
```bash
go run ./cmd/main.go

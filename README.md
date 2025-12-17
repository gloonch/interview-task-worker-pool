# Worker Pool
A minimal Go HTTP API to manage tasks using an in-memory, concurrency-safe store.

# Implementation Phases
- API endpoints + clean layering (store/service/http) + graceful shutdown.
- Worker pool + configurable workers + bounded queue + status transitions + overflow handling

## Requirements
- Go 1.25.5+

## Project Structure
- `cmd/server/main.go` — Wiring + HTTP server + graceful shutdown
- `internal/config` — Runtime config (port, workers, pool size, shutdown timeout)
- `internal/domain` — Task
- `internal/store/memory` — In-memory task store (map + RWMutex, incremental int64 ID)
- `internal/service` — Use-cases + validation + error mapping
- `internal/http/handlers` — Endpoints
- `internal/router` — Routes using `net/http` patterns (Go 1.22+ style)

## Worker Pool Behavior
- **Enqueue on create**
    - `POST /tasks` creates a task in the in-memory store with status `pending`.
    - The created task ID is then enqueued into the worker pool queue.

- **Pending pool**
    - The pending pool is an in-memory **buffered channel** with capacity `POOL_SIZE` (default: `10`).
    - This channel acts as the single source of “pending work” for workers.

- **Workers**
    - A fixed number of worker goroutines (count = `WORKERS`, default: `5`) are started at boot.

- **Task processing**
    - When a worker receives a task ID from the queue:
        - reads the task from the shared store
        - updates status to `running`
        - sleeps for the task’s `WorkDuration` (randomized at creation time, 1–5 seconds)
        - updates status to `done`
    - Key events are logged:
        - task started (planned duration)
        - task completed (actual elapsed vs planned time)

- **Overflow / backpressure**
    - If the queue is full:
        - the task is marked as `failed`
        - `task.Error` is set to `"task pool is full"`
        - the API responds with `503 Service Unavailable`

- **Shutdown behavior**
    - On SIGINT/SIGTERM:
        - the HTTP server stops accepting new requests
        - the queue is closed
        - workers drain remaining queued tasks and exit
        - shutdown is best-effort within `SHUTDOWN_TIMEOUT`


## HTTP Status Codes

- **201 Created**
    - Task created successfully.
    - Response body returns the created task.

- **400 Bad Request**
    - Invalid JSON payload
    - Invalid input (e.g. empty `title`)
    - Invalid path parameter (e.g., non-numeric or `id <= 0`)

- **404 Not Found**
    - Task with the given `{id}` does not exist.

- **503 Service Unavailable**
    - Worker pool queue is full (backpressure): task is marked as `failed` with `error="task pool is full"`.
    - Worker pool is closed (during shutdown): task is marked as `failed` with `error="task pool is closed"`.


## Configuration (.env)
This project loads configuration from a `.env` file using `godotenv` (or from actual environment variables).


## Run locally
```bash
go run ./cmd/main.go
```

## Run with Docker
Build:
```bash
docker build -t interview-task-worker-pool .
```
Run (loads configuration from .env):
```bash
docker run --rm -p 8080:8080 --env-file .env interview-task-worker-pool
```



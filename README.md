# TaskTide

Distributed workflow platform — design DAGs in a drag-and-drop UI and execute them across a containerized fleet of workers, with Postgres as the source of truth and S3-compatible storage for file payloads.

## Features

- **Visual DAG builder** — drag-and-drop workflow editor (ReactFlow) with task palette, dependency edges, and live execution monitor
- **Distributed execution** — three services (API, queue/scheduler, worker) scaled independently; workers self-register and receive tasks over HTTP
- **Dependency-aware scheduling** — topological sort at launch plus unblocked-task queries; queue assigns ready tasks to free workers and re-triggers on every completion
- **File dataflow** — tasks pass data via files in MinIO/S3; chained outputs feed downstream inputs automatically
- **Durable state** — workflow/task executions and per-task attempts persisted in Postgres (sqlc + goose migrations)
- **Extensible task registry** — `NOOP`, `OUTPUT`, `csv.read`, `doc.read` executors included; new types plug into one switch
- **One-command deploy** — Docker Compose: Postgres → migrations → API/queue/worker → MinIO → nginx-served frontend

## Tech Stack

- **Backend:** Go 1.26, Chi, pgx, sqlc, goose, Postgres 15, MinIO (S3)
- **Frontend:** React 19, Vite, ReactFlow (@xyflow), Tailwind CSS 4
- **Infra:** Docker multi-stage builds + Compose, nginx

## Getting Started

### Prerequisites

- Docker + Docker Compose (full stack), or Go 1.26 + Node 18+ for local dev

### Run (full stack)

```bash
docker compose up --build
```

- App: `http://localhost:3000`
- API: `http://localhost:8080`
- MinIO console: `http://localhost:9001`

### Run (local dev)

```bash
# Backend (needs Postgres + MinIO running; copy required env vars into .env)
go mod download
go run ./cmd/api     # API service
go run ./cmd/queue   # scheduler service
go run ./cmd/worker  # worker (self-registers with the queue)

# Frontend
cd frontend && npm install && npm run dev
```

Required env: `POSTGRES_USERNAME`, `POSTGRES_PASSWORD`, `POSTGRES_DBSTRING`, `MINIO_USER`, `MINIO_PASSWORD`, `MINIO_HOST`, `MINIO_PORT`, `FILES_BUCKET_NAME`, `API_HOST/PORT`, `QUEUE_HOST/PORT`, `WORKER_HOST/PORT`.

### Try it

1. Open the UI, create a workflow, and drop tasks onto the canvas.
2. Upload an input file, chain tasks (`A → B → C → OUTPUT`), and start the run.
3. Watch per-task status update in the execution monitor.
4. Or run the scripted demo: `go run ./scripts/endToEnd.go`

## API Reference (selected)

**API service** — `POST /workflow/create`, `POST /workflow/add`, `POST /workflow/dependency`, `POST /workflow/` (start), `POST /workflow/status`, `GET /workflow/list`, `POST /file` (upload), `GET /file?id=`
**Queue service** — `POST /` (enqueue), `GET /` (poll/assign), `POST /worker` (register), `POST /worker/completion`
**Worker service** — `POST /task` (execute asynchronously)

## Project Structure

```
cmd/{api,queue,worker}/   # Service entrypoints
internal/
├── api/                  # HTTP routers per service
├── service/              # Business logic (startWorkflow, assign, execute)
├── repositories/         # Postgres + MinIO access layer
├── tasks/                # Task executors (noop, output, csv, doc)
├── models/               # Workflow/task domain types
└── db/                   # sqlc-generated queries
queries/                  # SQL sources for sqlc
migrations/               # goose migrations
frontend/src/             # ReactFlow builder + execution monitor
```

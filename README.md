# Concurrent Task Queue with Monitoring Dashboard

A background task processing platform with priority scheduling, delayed execution, automatic retries with exponential backoff, and dead-letter handling. Built with Go, Next.js, and Redis.

## Architecture

```
┌──────────────┐     ┌──────────────────┐     ┌───────────┐
│   Next.js    │────▶│   Go Backend     │────▶│   Redis   │
│  Dashboard   │     │  (Worker Pool)   │     │           │
│  :3000       │     │  :8080           │     │  :6379    │
└──────────────┘     └──────────────────┘     └───────────┘
```

**Go backend**: HTTP API + worker pool with goroutines polling a Redis sorted set.
**Next.js frontend**: 3-panel dashboard (submit tasks, view metrics, view dead-letter queue).
**Redis**: priority queue (sorted set), delayed queue (sorted set), dead-letter list, metrics hash.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)

That's it — Go and Node.js run inside containers.

## Quick Start

```bash
# Clone and start
cd weekend_project_1
docker-compose up --build
```

Services will be available at:
- **Dashboard**: http://localhost:3000
- **API**: http://localhost:8080
- **Redis**: localhost:6379

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/tasks` | Submit a task |
| `GET` | `/api/metrics` | Get processing metrics |
| `GET` | `/api/tasks/failed?offset=0&limit=20` | Get dead-lettered tasks |
| `GET` | `/api/health` | Health check |

### Submit a task

```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"id":"test-1","priority":5,"delay":0,"max_retries":3}'
```

**Fields:**
- `id` (string, required): Unique task identifier
- `priority` (int): 1-10, higher priority processes first
- `delay` (int): Seconds to wait before task becomes eligible
- `max_retries` (int): Number of retry attempts before dead-lettering

## How It Works

1. **Submit**: Tasks enter the priority queue (or delayed queue if `delay > 0`)
2. **Schedule**: A scheduler polls every 1s, promoting due delayed tasks to the ready queue
3. **Execute**: Worker goroutines poll the ready queue (ZPOPMIN), highest priority first
4. **Retry**: ~30% simulated failure rate. Failed tasks retry with exponential backoff: `min(2^retries * 1s, 60s)`
5. **Dead-letter**: Tasks exhausting retries go to the dead-letter list
6. **Monitor**: Dashboard auto-refreshes metrics (3s) and failed tasks (5s)

## Configuration

Environment variables (set in `docker-compose.yml`):

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `SERVER_PORT` | `8080` | API server port |
| `WORKER_COUNT` | `5` | Number of worker goroutines |
| `POLL_INTERVAL_MS` | `500` | Worker poll interval in ms |

## Stopping

```bash
# Graceful shutdown (workers finish in-progress tasks)
docker-compose down

# Remove volumes too
docker-compose down -v
```

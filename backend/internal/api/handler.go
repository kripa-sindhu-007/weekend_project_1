package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/weekend-project/taskqueue/internal/model"
	"github.com/weekend-project/taskqueue/internal/queue"
	"github.com/weekend-project/taskqueue/internal/store"
	"github.com/weekend-project/taskqueue/internal/worker"
)

type Handler struct {
	queue      *queue.PriorityQueue
	delayed    *queue.DelayedScheduler
	deadLetter *store.DeadLetterStore
	metrics    *store.MetricsStore
	pool       *worker.Pool
	redis      *redis.Client
	events     *store.EventStore
	workerState *store.WorkerStateStore
	queuePeek  *store.QueuePeekStore
}

func NewHandler(
	q *queue.PriorityQueue,
	d *queue.DelayedScheduler,
	dl *store.DeadLetterStore,
	m *store.MetricsStore,
	p *worker.Pool,
	r *redis.Client,
	events *store.EventStore,
	workerState *store.WorkerStateStore,
	queuePeek *store.QueuePeekStore,
) *Handler {
	return &Handler{
		queue:       q,
		delayed:     d,
		deadLetter:  dl,
		metrics:     m,
		pool:        p,
		redis:       r,
		events:      events,
		workerState: workerState,
		queuePeek:   queuePeek,
	}
}

type submitRequest struct {
	ID         string `json:"id"`
	Priority   int    `json:"priority"`
	Delay      int    `json:"delay"`
	MaxRetries int    `json:"max_retries"`
}

func (h *Handler) SubmitTask(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if req.ID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	task := model.Task{
		ID:         req.ID,
		Priority:   req.Priority,
		Delay:      req.Delay,
		MaxRetries: req.MaxRetries,
		Status:     model.StatusPending,
		CreatedAt:  time.Now(),
	}

	ctx := r.Context()

	// Emit submitted event
	event := model.TaskEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		TaskID:    task.ID,
		Type:      "submitted",
		WorkerID:  -1,
		Detail:    fmt.Sprintf("Priority=%d, Delay=%ds, MaxRetries=%d", task.Priority, task.Delay, task.MaxRetries),
		Timestamp: time.Now(),
	}
	h.events.Push(ctx, event)
	h.metrics.IncrSubmitted(ctx)

	if req.Delay > 0 {
		delay := time.Duration(req.Delay) * time.Second
		if err := h.delayed.Schedule(ctx, task, delay); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	} else {
		if err := h.queue.Enqueue(ctx, task); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queueSize, _ := h.queue.Size(ctx)
	activeWorkers := h.pool.ActiveWorkers()

	m, err := h.metrics.Get(ctx, queueSize, activeWorkers)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) GetFailedTasks(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if limit <= 0 {
		limit = 20
	}

	tasks, err := h.deadLetter.List(r.Context(), offset, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.redis.Ping(r.Context()).Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// --- New Endpoints ---

func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if limit <= 0 {
		limit = 50
	}
	events, err := h.events.List(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handler) GetWorkers(w http.ResponseWriter, r *http.Request) {
	states, err := h.workerState.GetAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, states)
}

func (h *Handler) GetQueues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ready, err := h.queuePeek.PeekReady(ctx, 20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	delayed, err := h.queuePeek.PeekDelayed(ctx, 20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ready":   ready,
		"delayed": delayed,
	})
}

func (h *Handler) GetEnhancedMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queueSize, _ := h.queue.Size(ctx)
	activeWorkers := h.pool.ActiveWorkers()
	delayedSize, _ := h.queuePeek.DelayedSize(ctx)
	deadLetterSize, _ := h.queuePeek.DeadLetterSize(ctx)

	m, err := h.metrics.GetEnhanced(ctx, queueSize, activeWorkers, delayedSize, deadLetterSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

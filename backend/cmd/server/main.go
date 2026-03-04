package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/weekend-project/taskqueue/internal/api"
	"github.com/weekend-project/taskqueue/internal/config"
	"github.com/weekend-project/taskqueue/internal/queue"
	"github.com/weekend-project/taskqueue/internal/store"
	"github.com/weekend-project/taskqueue/internal/worker"
)

func main() {
	cfg := config.Load()

	// Redis
	redisClient := store.NewRedisClient(cfg.RedisAddr, cfg.RedisPass)
	defer redisClient.Close()

	// Stores
	deadLetterStore := store.NewDeadLetterStore(redisClient)
	metricsStore := store.NewMetricsStore(redisClient)
	eventStore := store.NewEventStore(redisClient)
	workerStateStore := store.NewWorkerStateStore(redisClient)
	queuePeekStore := store.NewQueuePeekStore(redisClient)

	// Queue
	priorityQueue := queue.NewPriorityQueue(redisClient)
	delayedScheduler := queue.NewDelayedScheduler(redisClient, priorityQueue, eventStore)

	// Workers
	executor := worker.NewExecutor(delayedScheduler, deadLetterStore, metricsStore, eventStore, workerStateStore)
	pool := worker.NewPool(priorityQueue, executor, cfg.WorkerCount, cfg.PollInterval, workerStateStore)

	// API
	handler := api.NewHandler(
		priorityQueue, delayedScheduler, deadLetterStore, metricsStore,
		pool, redisClient, eventStore, workerStateStore, queuePeekStore,
	)
	router := api.NewRouter(handler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: router,
	}

	// Graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start delayed scheduler
	go delayedScheduler.Start(ctx)

	// Start worker pool
	pool.Start(ctx)

	// Start HTTP server
	go func() {
		log.Printf("Server listening on :%s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutdown signal received")

	// Shutdown HTTP server with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Wait for workers to finish in-progress tasks
	pool.Wait()
	log.Println("Shutdown complete")
}

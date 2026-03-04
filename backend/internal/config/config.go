package config

import (
	"os"
	"strconv"
)

type Config struct {
	RedisAddr    string
	RedisPass    string
	ServerPort   string
	WorkerCount  int
	PollInterval int // milliseconds
}

func Load() *Config {
	return &Config{
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:    getEnv("REDIS_PASSWORD", ""),
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		WorkerCount:  getEnvInt("WORKER_COUNT", 5),
		PollInterval: getEnvInt("POLL_INTERVAL_MS", 500),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

package main

import (
	"log/slog"
	"os"

	"github.com/lcrownover/paste/internal/server"
)

func main() {
	httpAddr := getEnv("HTTP_ADDR", "0.0.0.0:3000")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")

	slog.Info("connection info", "httpAddr", httpAddr, "redisAddr", redisAddr)

	server, err := server.NewServer(httpAddr, redisAddr)
	if err != nil {
		slog.Error("failed to create new server instance", "error", err)
		os.Exit(1)
	}

	slog.Info("Paste server starting", "addr", httpAddr)
	err = server.Serve()
	if err != nil {
		slog.Error("Error starting server", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

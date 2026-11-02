package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MayurJ1609/common-lib/config"
	"github.com/MayurJ1609/common-lib/db"
	"github.com/MayurJ1609/common-lib/health"
	"github.com/MayurJ1609/common-lib/httpx"
	"github.com/MayurJ1609/common-lib/logger"
	"github.com/MayurJ1609/common-lib/metrics"
	"github.com/MayurJ1609/common-lib/server"
	"github.com/MayurJ1609/common-lib/tracing"
	"github.com/example/go-boilerplate/internal/domain"
	"github.com/example/go-boilerplate/internal/handler/rest"
	mongostore "github.com/example/go-boilerplate/internal/repository/mongo"
	postgresrepo "github.com/example/go-boilerplate/internal/repository/postgres"
)

func main() {
	log := logger.New(logger.Options{Service: "go-boilerplate", Level: envOrDefault("LOG_LEVEL", "info")})
	slog.SetDefault(log)
	if err := run(log); err != nil {
		log.Error("REST API stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) (result error) {
	cfg, err := config.LoadDependencyConfig()
	if err != nil {
		return fmt.Errorf("load dependency configuration: %w", err)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
	clients, err := db.Connect(connectCtx, cfg)
	connectCancel()
	if err != nil {
		return fmt.Errorf("connect dependencies: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := clients.Close(closeCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("close dependencies: %w", err))
		}
	}()

	metricRegistry, err := metrics.NewRegistry("go_boilerplate")
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}
	if err := metrics.RegisterPostgresPool(metricRegistry.Prometheus, clients.Postgres); err != nil {
		return fmt.Errorf("register PostgreSQL metrics: %w", err)
	}
	traceProvider, err := tracing.New(context.Background(), tracing.Options{
		ServiceName:    "go-boilerplate",
		ServiceVersion: "dev",
		Endpoint:       envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		Insecure:       true,
	})
	if err != nil {
		return fmt.Errorf("initialize tracing: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := traceProvider.Shutdown(shutdownCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("shutdown tracing: %w", err))
		}
	}()

	mux := http.NewServeMux()
	health.New(map[string]health.Check{"dependencies": clients.Ping}, 3*time.Second).Register(mux)
	mux.Handle("GET /metrics", metricRegistry.Handler())
	mux.Handle("/", rest.NewHandlerWithStores(
		domain.Greeter{},
		clients,
		clients,
		postgresrepo.NewItems(clients.Postgres),
		mongostore.NewItems(clients.MongoDatabase()),
	))

	handler := httpx.Chain(
		mux,
		httpx.RequestIDMiddleware,
		httpx.LoggingMiddleware(log),
		httpx.MetricsMiddleware(metricRegistry),
		httpx.TracingMiddleware(),
		httpx.RecoverMiddleware(log),
	)
	httpServer := &http.Server{
		Addr:              envOrDefault("HTTP_ADDR", ":8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("REST API listening", "addr", httpServer.Addr)
	if err := (server.Server{HTTP: httpServer, ShutdownTimeout: 10 * time.Second}).Run(ctx); err != nil {
		return fmt.Errorf("run HTTP server: %w", err)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-boilerplate/internal/platform/dependencies"
	"github.com/example/go-boilerplate/internal/service"
	mongostore "github.com/example/go-boilerplate/internal/storage/mongo"
	"github.com/example/go-boilerplate/internal/storage/postgres"
	"github.com/example/go-boilerplate/internal/transport/rest"
)

func main() {
	config, err := dependencies.LoadConfig()
	if err != nil {
		log.Fatalf("load dependency configuration: %v", err)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
	clients, err := dependencies.Connect(connectCtx, config)
	connectCancel()
	if err != nil {
		log.Fatalf("connect dependencies: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := clients.Close(closeCtx); err != nil {
			log.Printf("close dependencies: %v", err)
		}
	}()

	addr := envOrDefault("HTTP_ADDR", ":8080")
	server := &http.Server{
		Addr: addr,
		Handler: rest.NewHandlerWithStores(
			service.Greeter{},
			clients,
			clients,
			clients.Ping,
			postgres.NewItems(clients.Postgres),
			mongostore.NewItems(clients.MongoDatabase()),
		),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown: %v", err)
		}
	}()

	log.Printf("REST API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("REST API: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

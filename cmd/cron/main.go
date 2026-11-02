package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MayurJ1609/common-lib/config"
	"github.com/MayurJ1609/common-lib/db"
	"github.com/MayurJ1609/common-lib/logger"
	"github.com/example/go-boilerplate/internal/domain"
	"github.com/robfig/cron/v3"
)

func main() {
	log := logger.New(logger.Options{Service: "go-boilerplate", Level: envOrDefault("LOG_LEVEL", "info")})
	if err := run(log); err != nil {
		log.Error("cron runner stopped with an error", "error", err)
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

	spec := envOrDefault("CRON_SPEC", "@every 1m")
	scheduler := cron.New()
	if _, err := scheduler.AddFunc(spec, func() {
		log.Info("scheduled job", "message", domain.Greeter{}.Greet("scheduled job"))
	}); err != nil {
		return fmt.Errorf("invalid CRON_SPEC %q: %w", spec, err)
	}

	scheduler.Start()
	log.Info("cron runner started", "schedule", spec)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	<-signals
	<-scheduler.Stop().Done()
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

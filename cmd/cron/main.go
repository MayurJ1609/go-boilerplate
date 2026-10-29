package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-boilerplate/internal/config"
	"github.com/example/go-boilerplate/internal/database"
	"github.com/example/go-boilerplate/internal/domain"
	"github.com/robfig/cron/v3"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load dependency configuration: %v", err)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
	clients, err := database.Connect(connectCtx, cfg)
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

	spec := envOrDefault("CRON_SPEC", "@every 1m")
	scheduler := cron.New()
	if _, err := scheduler.AddFunc(spec, func() {
		log.Print(domain.Greeter{}.Greet("scheduled job"))
	}); err != nil {
		log.Fatalf("invalid CRON_SPEC %q: %v", spec, err)
	}

	scheduler.Start()
	log.Printf("cron runner started with schedule %q", spec)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	<-scheduler.Stop().Done()
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

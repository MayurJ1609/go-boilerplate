package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestRunRejectsInvalidConfiguration(t *testing.T) {
	t.Setenv("REDIS_DB", "not-a-number")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := run(log); err == nil || !strings.Contains(err.Error(), "load dependency configuration") {
		t.Fatalf("run() error = %v, want configuration error", err)
	}
}

func TestRunReturnsDependencyConnectionError(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "invalid-dsn")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := run(log); err == nil || !strings.Contains(err.Error(), "connect dependencies") {
		t.Fatalf("run() error = %v, want dependency connection error", err)
	}
}

func TestHTTPAddressEnvironmentDefault(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	if got := envOrDefault("HTTP_ADDR", ":8080"); got != ":8080" {
		t.Fatalf("default address = %q, want :8080", got)
	}
	t.Setenv("HTTP_ADDR", ":8181")
	if got := envOrDefault("HTTP_ADDR", ":8080"); got != ":8181" {
		t.Fatalf("configured address = %q, want :8181", got)
	}
}

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

func TestGRPCAddressEnvironmentDefault(t *testing.T) {
	t.Setenv("GRPC_ADDR", "")
	if got := envOrDefault("GRPC_ADDR", ":9090"); got != ":9090" {
		t.Fatalf("default address = %q, want :9090", got)
	}
	t.Setenv("GRPC_ADDR", ":9191")
	if got := envOrDefault("GRPC_ADDR", ":9090"); got != ":9191" {
		t.Fatalf("configured address = %q, want :9191", got)
	}
}

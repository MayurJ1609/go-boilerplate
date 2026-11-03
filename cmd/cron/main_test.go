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

func TestCronScheduleEnvironmentDefault(t *testing.T) {
	t.Setenv("CRON_SPEC", "")
	if got := envOrDefault("CRON_SPEC", "@every 1m"); got != "@every 1m" {
		t.Fatalf("default schedule = %q, want @every 1m", got)
	}
	t.Setenv("CRON_SPEC", "@every 30s")
	if got := envOrDefault("CRON_SPEC", "@every 1m"); got != "@every 30s" {
		t.Fatalf("configured schedule = %q, want @every 30s", got)
	}
}

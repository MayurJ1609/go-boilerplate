package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-boilerplate/internal/service"
)

func TestGreetEndpoint(t *testing.T) {
	handler := NewHandler(service.Greeter{})
	req := httptest.NewRequest(http.MethodGet, "/v1/greet?name=Ada", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body["message"], "Hello, Ada!"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestHealthEndpoint(t *testing.T) {
	handler := NewHandler(service.Greeter{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestReadinessEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		ready      func(context.Context) error
		wantStatus int
	}{
		{
			name:       "ready",
			ready:      func(context.Context) error { return nil },
			wantStatus: http.StatusNoContent,
		},
		{
			name: "dependency unavailable",
			ready: func(context.Context) error {
				return context.DeadlineExceeded
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandlerWithReadiness(service.Greeter{}, tt.ready)
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

type testCache struct {
	values map[string]string
}

func (c *testCache) GetCache(_ context.Context, key string) (string, bool, error) {
	value, found := c.values[key]
	return value, found, nil
}

func (c *testCache) SetCache(_ context.Context, key, value string, _ time.Duration) error {
	if c.values == nil {
		c.values = make(map[string]string)
	}
	c.values[key] = value
	return nil
}

type testPublisher struct {
	key   []byte
	value []byte
}

func (p *testPublisher) Publish(_ context.Context, key, value []byte) error {
	p.key = key
	p.value = value
	return nil
}

func TestGreetUsesCacheAndPublishesEvent(t *testing.T) {
	cache := &testCache{values: map[string]string{"greeting:Ada": "Hello, Ada!"}}
	publisher := &testPublisher{}
	handler := NewHandlerWithDependencies(service.Greeter{}, cache, publisher, func(context.Context) error { return nil })
	req := httptest.NewRequest(http.MethodGet, "/v1/greet?name=Ada", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := string(publisher.key), "Ada"; got != want {
		t.Fatalf("event key = %q, want %q", got, want)
	}
	var event map[string]string
	if err := json.Unmarshal(publisher.value, &event); err != nil {
		t.Fatalf("decode published event: %v", err)
	}
	if got, want := event["message"], "Hello, Ada!"; got != want {
		t.Fatalf("event message = %q, want %q", got, want)
	}
}

func TestGreetReturnsUnavailableWhenCacheFails(t *testing.T) {
	handler := NewHandlerWithDependencies(service.Greeter{}, failingCache{}, &testPublisher{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/greet?name=Ada", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

type failingCache struct{}

func (failingCache) GetCache(context.Context, string) (string, bool, error) {
	return "", false, errors.New("cache unavailable")
}

func (failingCache) SetCache(context.Context, string, string, time.Duration) error {
	return errors.New("cache unavailable")
}

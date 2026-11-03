package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-boilerplate/internal/domain"
)

func TestGreetEndpoint(t *testing.T) {
	handler := NewHandler(domain.Greeter{})
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

type testCache struct {
	values  map[string]string
	setErr  error
	getErr  error
	setKeys []string
}

func (c *testCache) GetCache(_ context.Context, key string) (string, bool, error) {
	if c.getErr != nil {
		return "", false, c.getErr
	}
	value, found := c.values[key]
	return value, found, nil
}

func (c *testCache) SetCache(_ context.Context, key, value string, ttl time.Duration) error {
	if ttl != 5*time.Minute {
		return errors.New("unexpected cache TTL")
	}
	c.setKeys = append(c.setKeys, key)
	if c.setErr != nil {
		return c.setErr
	}
	if c.values == nil {
		c.values = make(map[string]string)
	}
	c.values[key] = value
	return nil
}

type testPublisher struct {
	key    []byte
	value  []byte
	pubErr error
}

func (p *testPublisher) Publish(_ context.Context, key, value []byte) error {
	if p.pubErr != nil {
		return p.pubErr
	}
	p.key = key
	p.value = value
	return nil
}

func TestGreetUsesCacheAndPublishesEvent(t *testing.T) {
	cache := &testCache{values: map[string]string{"greeting:Ada": "Hello, Ada!"}}
	publisher := &testPublisher{}
	handler := NewHandlerWithDependencies(domain.Greeter{}, cache, publisher)
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
	handler := NewHandlerWithDependencies(domain.Greeter{}, failingCache{}, &testPublisher{})
	req := httptest.NewRequest(http.MethodGet, "/v1/greet?name=Ada", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestGreetCacheMissStoresGreetingAndUsesDefaultName(t *testing.T) {
	cache := &testCache{}
	handler := NewHandlerWithDependencies(domain.Greeter{}, cache, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/greet?name=%20%20", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if cache.values["greeting:World"] != "Hello, World!" {
		t.Fatalf("cached values = %#v, want World greeting", cache.values)
	}
	if len(cache.setKeys) != 1 || cache.setKeys[0] != "greeting:World" {
		t.Fatalf("cache keys = %v, want greeting:World", cache.setKeys)
	}
}

func TestGreetReturnsUnavailableWhenCacheWriteFails(t *testing.T) {
	handler := NewHandlerWithDependencies(domain.Greeter{}, &testCache{setErr: errors.New("cache write failed")}, nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/greet", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestGreetReturnsUnavailableWhenPublishFails(t *testing.T) {
	publisher := &testPublisher{pubErr: errors.New("broker unavailable")}
	handler := NewHandlerWithDependencies(domain.Greeter{}, nil, publisher)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/greet?name=Ada", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestGreetHandlesResponseWriteFailure(t *testing.T) {
	handler := NewHandler(domain.Greeter{})
	handler.ServeHTTP(failedResponseWriter{}, httptest.NewRequest(http.MethodGet, "/v1/greet?name=Ada", nil))
}

func TestWriteJSONLogsResponseWriteFailure(t *testing.T) {
	writeJSON(failedResponseWriter{}, http.StatusOK, map[string]string{"value": "test"})
}

type failingCache struct{}

func (failingCache) GetCache(context.Context, string) (string, bool, error) {
	return "", false, errors.New("cache unavailable")
}

func (failingCache) SetCache(context.Context, string, string, time.Duration) error {
	return errors.New("cache unavailable")
}

type failedResponseWriter struct{}

func (failedResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (failedResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("client disconnected")
}

func (failedResponseWriter) WriteHeader(int) {}

package rest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-boilerplate/internal/domain"
)

type Handler struct {
	greeter       domain.Greeter
	ready         func(context.Context) error
	cache         Cache
	publisher     Publisher
	postgresItems domain.ItemRepository
	mongoItems    domain.ItemRepository
}

type Cache interface {
	GetCache(context.Context, string) (string, bool, error)
	SetCache(context.Context, string, string, time.Duration) error
}

type Publisher interface {
	Publish(context.Context, []byte, []byte) error
}

func NewHandler(greeter domain.Greeter) http.Handler {
	return newHandler(greeter, nil, nil, nil)
}

func NewHandlerWithReadiness(greeter domain.Greeter, ready func(context.Context) error) http.Handler {
	return newHandler(greeter, ready, nil, nil)
}

func NewHandlerWithDependencies(greeter domain.Greeter, cache Cache, publisher Publisher, ready func(context.Context) error) http.Handler {
	return newHandler(greeter, ready, cache, publisher)
}

func NewHandlerWithStores(
	greeter domain.Greeter,
	cache Cache,
	publisher Publisher,
	ready func(context.Context) error,
	postgresItems domain.ItemRepository,
	mongoItems domain.ItemRepository,
) http.Handler {
	return newHandlerWithStores(greeter, ready, cache, publisher, postgresItems, mongoItems)
}

func newHandler(greeter domain.Greeter, ready func(context.Context) error, cache Cache, publisher Publisher) http.Handler {
	return newHandlerWithStores(greeter, ready, cache, publisher, nil, nil)
}

func newHandlerWithStores(
	greeter domain.Greeter,
	ready func(context.Context) error,
	cache Cache,
	publisher Publisher,
	postgresItems domain.ItemRepository,
	mongoItems domain.ItemRepository,
) http.Handler {
	h := Handler{
		greeter:       greeter,
		ready:         ready,
		cache:         cache,
		publisher:     publisher,
		postgresItems: postgresItems,
		mongoItems:    mongoItems,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /readyz", h.readiness)
	mux.HandleFunc("GET /v1/greet", h.greet)
	registerItemRoutes(mux, "/v1/postgres/items", h.postgresItems)
	registerItemRoutes(mux, "/v1/mongo/items", h.mongoItems)
	return mux
}

func (h Handler) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) readiness(w http.ResponseWriter, r *http.Request) {
	if h.ready == nil {
		http.Error(w, "dependencies are not configured", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := h.ready(ctx); err != nil {
		log.Printf("readiness check failed: %v", err)
		http.Error(w, "one or more dependencies are unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) greet(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "World"
	}
	cacheKey := "greeting:" + name
	message := ""
	if h.cache != nil {
		cached, found, err := h.cache.GetCache(r.Context(), cacheKey)
		if err != nil {
			log.Printf("read greeting cache: %v", err)
			http.Error(w, "greeting cache unavailable", http.StatusServiceUnavailable)
			return
		}
		if found {
			message = cached
		}
	}
	if message == "" {
		message = h.greeter.Greet(name)
		if h.cache != nil {
			if err := h.cache.SetCache(r.Context(), cacheKey, message, 5*time.Minute); err != nil {
				log.Printf("write greeting cache: %v", err)
				http.Error(w, "greeting cache unavailable", http.StatusServiceUnavailable)
				return
			}
		}
	}
	if h.publisher != nil {
		event, err := json.Marshal(map[string]string{"name": name, "message": message})
		if err != nil {
			log.Printf("encode greeting event: %v", err)
			http.Error(w, "failed to encode greeting event", http.StatusInternalServerError)
			return
		}
		if err := h.publisher.Publish(r.Context(), []byte(name), event); err != nil {
			log.Printf("publish greeting event: %v", err)
			http.Error(w, "message broker unavailable", http.StatusServiceUnavailable)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

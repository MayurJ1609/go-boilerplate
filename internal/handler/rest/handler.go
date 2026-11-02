package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-boilerplate/internal/domain"
)

type Handler struct {
	greeter       domain.Greeter
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
	return newHandler(greeter, nil, nil)
}

func NewHandlerWithDependencies(greeter domain.Greeter, cache Cache, publisher Publisher) http.Handler {
	return newHandler(greeter, cache, publisher)
}

func NewHandlerWithStores(
	greeter domain.Greeter,
	cache Cache,
	publisher Publisher,
	postgresItems domain.ItemRepository,
	mongoItems domain.ItemRepository,
) http.Handler {
	return newHandlerWithStores(greeter, cache, publisher, postgresItems, mongoItems)
}

func newHandler(greeter domain.Greeter, cache Cache, publisher Publisher) http.Handler {
	return newHandlerWithStores(greeter, cache, publisher, nil, nil)
}

func newHandlerWithStores(
	greeter domain.Greeter,
	cache Cache,
	publisher Publisher,
	postgresItems domain.ItemRepository,
	mongoItems domain.ItemRepository,
) http.Handler {
	h := Handler{
		greeter:       greeter,
		cache:         cache,
		publisher:     publisher,
		postgresItems: postgresItems,
		mongoItems:    mongoItems,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/greet", h.greet)
	registerItemRoutes(mux, "/v1/postgres/items", h.postgresItems)
	registerItemRoutes(mux, "/v1/mongo/items", h.mongoItems)
	return mux
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
			slog.ErrorContext(r.Context(), "read greeting cache", "error", err)
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
				slog.ErrorContext(r.Context(), "write greeting cache", "error", err)
				http.Error(w, "greeting cache unavailable", http.StatusServiceUnavailable)
				return
			}
		}
	}
	if h.publisher != nil {
		event, err := json.Marshal(map[string]string{"name": name, "message": message})
		if err != nil {
			slog.ErrorContext(r.Context(), "encode greeting event", "error", err)
			http.Error(w, "failed to encode greeting event", http.StatusInternalServerError)
			return
		}
		if err := h.publisher.Publish(r.Context(), []byte(name), event); err != nil {
			slog.ErrorContext(r.Context(), "publish greeting event", "error", err)
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

package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-boilerplate/internal/model"
	"github.com/example/go-boilerplate/internal/storage"
	"github.com/google/uuid"
)

type itemHandler struct {
	store storage.ItemStore
}

type itemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func registerItemRoutes(mux *http.ServeMux, basePath string, store storage.ItemStore) {
	h := itemHandler{store: store}
	mux.HandleFunc("GET "+basePath, h.list)
	mux.HandleFunc("POST "+basePath, h.create)
	mux.HandleFunc("GET "+basePath+"/{id}", h.get)
	mux.HandleFunc("PUT "+basePath+"/{id}", h.update)
	mux.HandleFunc("DELETE "+basePath+"/{id}", h.delete)
}

func (h itemHandler) list(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	items, err := h.store.List(r.Context())
	if err != nil {
		h.internalError(w, "list items", err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h itemHandler) get(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	item, err := h.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleStoreError(w, "get item", err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h itemHandler) create(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	input, ok := decodeItemRequest(w, r)
	if !ok {
		return
	}
	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	item, err := h.store.Create(r.Context(), model.Item{
		ID:          uuid.NewString(),
		Name:        input.Name,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		h.internalError(w, "create item", err)
		return
	}
	w.Header().Set("Location", r.URL.Path+"/"+item.ID)
	writeJSON(w, http.StatusCreated, item)
}

func (h itemHandler) update(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	input, ok := decodeItemRequest(w, r)
	if !ok {
		return
	}
	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	item, err := h.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleStoreError(w, "get item for update", err)
		return
	}
	item.Name = input.Name
	item.Description = input.Description
	item.UpdatedAt = time.Now().UTC()
	item, err = h.store.Update(r.Context(), item)
	if err != nil {
		h.handleStoreError(w, "update item", err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h itemHandler) delete(w http.ResponseWriter, r *http.Request) {
	if !h.requireStore(w) {
		return
	}
	if err := h.store.Delete(r.Context(), r.PathValue("id")); err != nil {
		h.handleStoreError(w, "delete item", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h itemHandler) requireStore(w http.ResponseWriter) bool {
	if h.store != nil {
		return true
	}
	http.Error(w, "storage is not configured", http.StatusServiceUnavailable)
	return false
}

func (h itemHandler) handleStoreError(w http.ResponseWriter, operation string, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}
	h.internalError(w, operation, err)
}

func (h itemHandler) internalError(w http.ResponseWriter, operation string, err error) {
	log.Printf("%s: %v", operation, err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func decodeItemRequest(w http.ResponseWriter, r *http.Request) (itemRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input itemRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON body: %v", err), http.StatusBadRequest)
		return itemRequest{}, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		http.Error(w, "request body must contain exactly one JSON object", http.StatusBadRequest)
		return itemRequest{}, false
	}
	input.Name = strings.TrimSpace(input.Name)
	return input, true
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}

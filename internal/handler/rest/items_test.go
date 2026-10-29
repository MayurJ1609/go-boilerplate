package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-boilerplate/internal/domain"
)

func TestItemCRUDUsesIndependentStores(t *testing.T) {
	postgresStore := newMemoryItemStore()
	mongoStore := newMemoryItemStore()
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, nil, postgresStore, mongoStore)

	for _, tt := range []struct {
		name   string
		prefix string
		store  *memoryItemStore
		other  *memoryItemStore
	}{
		{name: "PostgreSQL", prefix: "/v1/postgres/items", store: postgresStore, other: mongoStore},
		{name: "MongoDB", prefix: "/v1/mongo/items", store: mongoStore, other: postgresStore},
	} {
		t.Run(tt.name, func(t *testing.T) {
			create := performJSONRequest(t, handler, http.MethodPost, tt.prefix, `{"name":"Sample","description":"before"}`)
			if create.Code != http.StatusCreated {
				t.Fatalf("create status = %d, want %d: %s", create.Code, http.StatusCreated, create.Body)
			}

			var created domain.Item
			if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
				t.Fatalf("decode created item: %v", err)
			}
			if created.ID == "" || created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
				t.Fatalf("created item missing generated fields: %+v", created)
			}

			get := performJSONRequest(t, handler, http.MethodGet, tt.prefix+"/"+created.ID, "")
			if get.Code != http.StatusOK {
				t.Fatalf("get status = %d, want %d: %s", get.Code, http.StatusOK, get.Body)
			}

			update := performJSONRequest(t, handler, http.MethodPut, tt.prefix+"/"+created.ID, `{"name":"Updated","description":"after"}`)
			if update.Code != http.StatusOK {
				t.Fatalf("update status = %d, want %d: %s", update.Code, http.StatusOK, update.Body)
			}
			var updated domain.Item
			if err := json.Unmarshal(update.Body.Bytes(), &updated); err != nil {
				t.Fatalf("decode updated item: %v", err)
			}
			if updated.Name != "Updated" || updated.Description != "after" || !updated.UpdatedAt.After(created.UpdatedAt) {
				t.Fatalf("updated item has unexpected values: %+v", updated)
			}

			list := performJSONRequest(t, handler, http.MethodGet, tt.prefix, "")
			if list.Code != http.StatusOK {
				t.Fatalf("list status = %d, want %d: %s", list.Code, http.StatusOK, list.Body)
			}
			var items []domain.Item
			if err := json.Unmarshal(list.Body.Bytes(), &items); err != nil {
				t.Fatalf("decode item list: %v", err)
			}
			if len(items) != 1 || items[0].ID != created.ID {
				t.Fatalf("listed items = %+v, want the created item", items)
			}

			if got := len(tt.other.items); got != 0 {
				t.Fatalf("other backend has %d items, want independent storage", got)
			}

			remove := performJSONRequest(t, handler, http.MethodDelete, tt.prefix+"/"+created.ID, "")
			if remove.Code != http.StatusNoContent {
				t.Fatalf("delete status = %d, want %d: %s", remove.Code, http.StatusNoContent, remove.Body)
			}
			missing := performJSONRequest(t, handler, http.MethodGet, tt.prefix+"/"+created.ID, "")
			if missing.Code != http.StatusNotFound {
				t.Fatalf("get deleted item status = %d, want %d", missing.Code, http.StatusNotFound)
			}
		})
	}
}

func TestCreateItemRejectsInvalidInput(t *testing.T) {
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, nil, newMemoryItemStore(), newMemoryItemStore())
	for _, body := range []string{
		`{"name":" "}`,
		`{"name":"valid","unexpected":true}`,
		`{"name":"first"} {"name":"second"}`,
	} {
		t.Run(fmt.Sprintf("body %q", body), func(t *testing.T) {
			response := performJSONRequest(t, handler, http.MethodPost, "/v1/postgres/items", body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body)
			}
		})
	}
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type memoryItemStore struct {
	items map[string]domain.Item
}

func newMemoryItemStore() *memoryItemStore {
	return &memoryItemStore{items: make(map[string]domain.Item)}
}

func (s *memoryItemStore) List(context.Context) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryItemStore) Get(_ context.Context, id string) (domain.Item, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.Item{}, fmt.Errorf("lookup: %w", domain.ErrNotFound)
	}
	return item, nil
}

func (s *memoryItemStore) Create(_ context.Context, item domain.Item) (domain.Item, error) {
	s.items[item.ID] = item
	return item, nil
}

func (s *memoryItemStore) Update(_ context.Context, item domain.Item) (domain.Item, error) {
	if _, ok := s.items[item.ID]; !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *memoryItemStore) Delete(_ context.Context, id string) error {
	if _, ok := s.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

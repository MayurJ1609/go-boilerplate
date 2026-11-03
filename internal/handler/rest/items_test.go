package rest

import (
	"context"
	"encoding/json"
	"errors"
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
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, postgresStore, mongoStore)

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
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, newMemoryItemStore(), newMemoryItemStore())
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

func TestItemRoutesReportStoreFailures(t *testing.T) {
	storeErr := errors.New("database unavailable")
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, failingItemStore{err: storeErr}, newMemoryItemStore())
	for _, tt := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/v1/postgres/items"},
		{method: http.MethodGet, path: "/v1/postgres/items/id"},
		{method: http.MethodPost, path: "/v1/postgres/items", body: `{"name":"item"}`},
		{method: http.MethodPut, path: "/v1/postgres/items/id", body: `{"name":"item"}`},
		{method: http.MethodDelete, path: "/v1/postgres/items/id"},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := performJSONRequest(t, handler, tt.method, tt.path, tt.body)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusInternalServerError, response.Body)
			}
		})
	}
}

func TestItemRoutesRequireConfiguredStore(t *testing.T) {
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, nil, nil)
	for _, tt := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/v1/postgres/items"},
		{method: http.MethodGet, path: "/v1/postgres/items/id"},
		{method: http.MethodPost, path: "/v1/postgres/items", body: `{"name":"item"}`},
		{method: http.MethodPut, path: "/v1/postgres/items/id", body: `{"name":"item"}`},
		{method: http.MethodDelete, path: "/v1/postgres/items/id"},
	} {
		response := performJSONRequest(t, handler, tt.method, tt.path, tt.body)
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, response.Code, http.StatusServiceUnavailable)
		}
	}
}

func TestUpdateAndDeleteItemReturnNotFound(t *testing.T) {
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, newMemoryItemStore(), newMemoryItemStore())
	update := performJSONRequest(t, handler, http.MethodPut, "/v1/postgres/items/missing", `{"name":"item"}`)
	if update.Code != http.StatusNotFound {
		t.Fatalf("update status = %d, want %d", update.Code, http.StatusNotFound)
	}
	remove := performJSONRequest(t, handler, http.MethodDelete, "/v1/postgres/items/missing", "")
	if remove.Code != http.StatusNotFound {
		t.Fatalf("delete status = %d, want %d", remove.Code, http.StatusNotFound)
	}
}

func TestUpdateItemReportsUpdateFailure(t *testing.T) {
	store := updateFailingItemStore{
		memoryItemStore: newMemoryItemStore(),
		err:             errors.New("database unavailable"),
	}
	store.items["item-1"] = domain.Item{ID: "item-1", Name: "old"}
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, store, newMemoryItemStore())
	response := performJSONRequest(t, handler, http.MethodPut, "/v1/postgres/items/item-1", `{"name":"new"}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestCreateItemRejectsMalformedAndOversizedJSON(t *testing.T) {
	handler := NewHandlerWithStores(domain.Greeter{}, nil, nil, newMemoryItemStore(), newMemoryItemStore())
	for _, body := range []string{
		`{"name":`,
		`{"name":"` + strings.Repeat("x", (1<<20)+1) + `"}`,
	} {
		response := performJSONRequest(t, handler, http.MethodPost, "/v1/postgres/items", body)
		if response.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	}
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type failingItemStore struct {
	err error
}

type updateFailingItemStore struct {
	*memoryItemStore
	err error
}

func (s updateFailingItemStore) Update(context.Context, domain.Item) (domain.Item, error) {
	return domain.Item{}, s.err
}

func (s failingItemStore) List(context.Context) ([]domain.Item, error) {
	return nil, s.err
}

func (s failingItemStore) Get(context.Context, string) (domain.Item, error) {
	return domain.Item{}, s.err
}

func (s failingItemStore) Create(context.Context, domain.Item) (domain.Item, error) {
	return domain.Item{}, s.err
}

func (s failingItemStore) Update(context.Context, domain.Item) (domain.Item, error) {
	return domain.Item{}, s.err
}

func (s failingItemStore) Delete(context.Context, string) error {
	return s.err
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

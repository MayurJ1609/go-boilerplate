package postgres

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/example/go-boilerplate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestItemsCRUD(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create pgx mock: %v", err)
	}
	defer mock.Close()
	store := NewItems(mock)
	now := time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)

	rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
		AddRow("item-1", "sample", "description", now, now)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, description, created_at, updated_at")).
		WillReturnRows(rows)
	items, err := store.List(context.Background())
	if err != nil || len(items) != 1 || items[0].ID != "item-1" {
		t.Fatalf("List() = %+v, %v; want item-1", items, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, description, created_at, updated_at")).
		WithArgs("item-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
			AddRow("item-1", "sample", "description", now, now))
	item, err := store.Get(context.Background(), "item-1")
	if err != nil || item.ID != "item-1" {
		t.Fatalf("Get() = %+v, %v; want item-1", item, err)
	}

	input := domain.Item{ID: "item-1", Name: "sample", Description: "description", CreatedAt: now, UpdatedAt: now}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO items")).
		WithArgs(input.ID, input.Name, input.Description, input.CreatedAt, input.UpdatedAt).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
			AddRow(input.ID, input.Name, input.Description, input.CreatedAt, input.UpdatedAt))
	if created, err := store.Create(context.Background(), input); err != nil || created.ID != input.ID {
		t.Fatalf("Create() = %+v, %v; want item-1", created, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE items")).
		WithArgs(input.ID, input.Name, input.Description, input.UpdatedAt).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
			AddRow(input.ID, input.Name, input.Description, input.CreatedAt, input.UpdatedAt))
	if updated, err := store.Update(context.Background(), input); err != nil || updated.ID != input.ID {
		t.Fatalf("Update() = %+v, %v; want item-1", updated, err)
	}

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM items WHERE id = $1")).
		WithArgs("item-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	if err := store.Delete(context.Background(), "item-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestItemsNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create pgx mock: %v", err)
	}
	defer mock.Close()
	store := NewItems(mock)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, description, created_at, updated_at")).
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)
	if _, err := store.Get(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE items")).
		WithArgs("missing", "name", "description", time.Time{}).
		WillReturnError(pgx.ErrNoRows)
	if _, err := store.Update(context.Background(), domain.Item{ID: "missing", Name: "name", Description: "description"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM items WHERE id = $1")).
		WithArgs("missing").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	if err := store.Delete(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestItemsDatabaseErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create pgx mock: %v", err)
	}
	defer mock.Close()
	store := NewItems(mock)
	dbErr := errors.New("database unavailable")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, description, created_at, updated_at")).WillReturnError(dbErr)
	if _, err := store.List(context.Background()); !errors.Is(err, dbErr) {
		t.Fatalf("List() error = %v, want wrapped database error", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, description, created_at, updated_at")).
		WithArgs("item-1").
		WillReturnError(dbErr)
	if _, err := store.Get(context.Background(), "item-1"); !errors.Is(err, dbErr) {
		t.Fatalf("Get() error = %v, want wrapped database error", err)
	}

	item := domain.Item{ID: "item-1"}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO items")).
		WithArgs(item.ID, item.Name, item.Description, item.CreatedAt, item.UpdatedAt).
		WillReturnError(dbErr)
	if _, err := store.Create(context.Background(), item); !errors.Is(err, dbErr) {
		t.Fatalf("Create() error = %v, want wrapped database error", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE items")).
		WithArgs(item.ID, item.Name, item.Description, item.UpdatedAt).
		WillReturnError(dbErr)
	if _, err := store.Update(context.Background(), item); !errors.Is(err, dbErr) {
		t.Fatalf("Update() error = %v, want wrapped database error", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM items WHERE id = $1")).
		WithArgs(item.ID).
		WillReturnError(dbErr)
	if err := store.Delete(context.Background(), item.ID); !errors.Is(err, dbErr) {
		t.Fatalf("Delete() error = %v, want wrapped database error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestItemsListScanAndIterationErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create pgx mock: %v", err)
	}
	defer mock.Close()
	store := NewItems(mock)
	columns := []string{"id", "name", "description", "created_at", "updated_at"}
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, description, created_at, updated_at")).
		WillReturnRows(pgxmock.NewRows(columns).AddRow("item-1", "name", "description", "not-a-time", now))
	if _, err := store.List(context.Background()); err == nil {
		t.Fatal("List() expected scan error")
	}

	rowErr := errors.New("row iteration failed")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, description, created_at, updated_at")).
		WillReturnRows(pgxmock.NewRows(columns).
			AddRow("item-1", "name", "description", now, now).
			RowError(0, rowErr))
	if _, err := store.List(context.Background()); !errors.Is(err, rowErr) {
		t.Fatalf("List() error = %v, want wrapped iteration error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

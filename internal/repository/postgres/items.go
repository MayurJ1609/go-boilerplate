package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-boilerplate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Items struct {
	pool *pgxpool.Pool
}

func NewItems(pool *pgxpool.Pool) *Items {
	return &Items{pool: pool}
}

func (s *Items) List(ctx context.Context) ([]domain.Item, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM items
		ORDER BY created_at, id
	`)
	if err != nil {
		return nil, fmt.Errorf("list PostgreSQL items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Item, 0)
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan PostgreSQL item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PostgreSQL items: %w", err)
	}
	return items, nil
}

func (s *Items) Get(ctx context.Context, id string) (domain.Item, error) {
	var item domain.Item
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM items
		WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Item{}, fmt.Errorf("get PostgreSQL item: %w", err)
	}
	return item, nil
}

func (s *Items) Create(ctx context.Context, item domain.Item) (domain.Item, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO items (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, created_at, updated_at
	`, item.ID, item.Name, item.Description, item.CreatedAt, item.UpdatedAt).
		Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.Item{}, fmt.Errorf("create PostgreSQL item: %w", err)
	}
	return item, nil
}

func (s *Items) Update(ctx context.Context, item domain.Item) (domain.Item, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE items
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1
		RETURNING id, name, description, created_at, updated_at
	`, item.ID, item.Name, item.Description, item.UpdatedAt).
		Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Item{}, fmt.Errorf("update PostgreSQL item: %w", err)
	}
	return item, nil
}

func (s *Items) Delete(ctx context.Context, id string) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete PostgreSQL item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

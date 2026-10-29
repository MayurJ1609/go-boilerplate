package storage

import (
	"context"
	"errors"

	"github.com/example/go-boilerplate/internal/model"
)

var ErrNotFound = errors.New("item not found")

type ItemStore interface {
	List(context.Context) ([]model.Item, error)
	Get(context.Context, string) (model.Item, error)
	Create(context.Context, model.Item) (model.Item, error)
	Update(context.Context, model.Item) (model.Item, error)
	Delete(context.Context, string) error
}

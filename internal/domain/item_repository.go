package domain

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("item not found")

type ItemRepository interface {
	List(context.Context) ([]Item, error)
	Get(context.Context, string) (Item, error)
	Create(context.Context, Item) (Item, error)
	Update(context.Context, Item) (Item, error)
	Delete(context.Context, string) error
}

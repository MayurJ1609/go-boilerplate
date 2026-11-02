package domain

import (
	"context"

	commonerrors "github.com/MayurJ1609/common-lib/errors"
)

var ErrNotFound = commonerrors.New(commonerrors.CodeNotFound, "item not found")

type ItemRepository interface {
	List(context.Context) ([]Item, error)
	Get(context.Context, string) (Item, error)
	Create(context.Context, Item) (Item, error)
	Update(context.Context, Item) (Item, error)
	Delete(context.Context, string) error
}

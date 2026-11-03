package mongostore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-boilerplate/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestItemsCRUD(t *testing.T) {
	mt := mtest.New(t)
	mt.RunOpts("CRUD", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
		store := NewItems(mt.DB)
		now := time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)
		document := bson.D{
			{Key: "_id", Value: "item-1"},
			{Key: "name", Value: "sample"},
			{Key: "description", Value: "description"},
			{Key: "created_at", Value: now},
			{Key: "updated_at", Value: now},
		}
		namespace := mt.DB.Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, namespace, mtest.FirstBatch, document))
		items, err := store.List(context.Background())
		if err != nil || len(items) != 1 || items[0].ID != "item-1" {
			mt.Fatalf("List() = %+v, %v; want item-1", items, err)
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(0, namespace, mtest.FirstBatch, document))
		item, err := store.Get(context.Background(), "item-1")
		if err != nil || item.ID != "item-1" {
			mt.Fatalf("Get() = %+v, %v; want item-1", item, err)
		}

		input := domain.Item{ID: "item-1", Name: "sample", Description: "description", CreatedAt: now, UpdatedAt: now}
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		if created, err := store.Create(context.Background(), input); err != nil || created.ID != input.ID {
			mt.Fatalf("Create() = %+v, %v; want item-1", created, err)
		}

		mt.AddMockResponses(
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateCursorResponse(0, namespace, mtest.FirstBatch, document),
		)
		if updated, err := store.Update(context.Background(), input); err != nil || updated.ID != input.ID {
			mt.Fatalf("Update() = %+v, %v; want item-1", updated, err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
		if err := store.Delete(context.Background(), "item-1"); err != nil {
			mt.Fatalf("Delete() error = %v", err)
		}
	})
}

func TestItemsNotFound(t *testing.T) {
	mt := mtest.New(t)
	mt.RunOpts("not found", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
		store := NewItems(mt.DB)
		namespace := mt.DB.Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, namespace, mtest.FirstBatch))
		if _, err := store.Get(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
			mt.Fatalf("Get() error = %v, want ErrNotFound", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}, bson.E{Key: "nModified", Value: 0}))
		if _, err := store.Update(context.Background(), domain.Item{ID: "missing"}); !errors.Is(err, domain.ErrNotFound) {
			mt.Fatalf("Update() error = %v, want ErrNotFound", err)
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}))
		if err := store.Delete(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
			mt.Fatalf("Delete() error = %v, want ErrNotFound", err)
		}
	})
}

func TestItemsListDecodeError(t *testing.T) {
	mt := mtest.New(t)
	mt.RunOpts("decode error", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
		store := NewItems(mt.DB)
		namespace := mt.DB.Name() + "." + mt.Coll.Name()
		invalidDocument := bson.D{{Key: "_id", Value: bson.D{{Key: "not", Value: "a string"}}}}
		mt.AddMockResponses(mtest.CreateCursorResponse(0, namespace, mtest.FirstBatch, invalidDocument))

		if _, err := store.List(context.Background()); err == nil {
			mt.Fatal("List() expected BSON decode error")
		}
	})
}

func TestItemsMongoErrors(t *testing.T) {
	mt := mtest.New(t)
	mt.RunOpts("database errors", mtest.NewOptions().ClientType(mtest.Mock), func(mt *mtest.T) {
		store := NewItems(mt.DB)
		serverErr := mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    123,
			Message: "database unavailable",
		})

		mt.AddMockResponses(serverErr)
		if _, err := store.List(context.Background()); err == nil {
			mt.Fatal("List() expected database error")
		}

		mt.AddMockResponses(serverErr)
		if _, err := store.Get(context.Background(), "item-1"); err == nil || errors.Is(err, mongo.ErrNoDocuments) {
			mt.Fatalf("Get() error = %v, want wrapped server error", err)
		}

		mt.AddMockResponses(serverErr)
		if _, err := store.Create(context.Background(), domain.Item{ID: "item-1"}); err == nil {
			mt.Fatal("Create() expected database error")
		}

		mt.AddMockResponses(serverErr)
		if _, err := store.Update(context.Background(), domain.Item{ID: "item-1"}); err == nil {
			mt.Fatal("Update() expected database error")
		}

		mt.AddMockResponses(serverErr)
		if err := store.Delete(context.Background(), "item-1"); err == nil {
			mt.Fatal("Delete() expected database error")
		}
	})
}

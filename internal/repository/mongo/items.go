package mongostore

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-boilerplate/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Items struct {
	collection *mongo.Collection
}

func NewItems(database *mongo.Database) *Items {
	return &Items{collection: database.Collection("items")}
}

func (s *Items) List(ctx context.Context) ([]domain.Item, error) {
	cursor, err := s.collection.Find(ctx, bson.D{}, nil)
	if err != nil {
		return nil, fmt.Errorf("list MongoDB items: %w", err)
	}
	defer cursor.Close(ctx)

	items := make([]domain.Item, 0)
	if err := cursor.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("decode MongoDB items: %w", err)
	}
	return items, nil
}

func (s *Items) Get(ctx context.Context, id string) (domain.Item, error) {
	var item domain.Item
	err := s.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&item)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.Item{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Item{}, fmt.Errorf("get MongoDB item: %w", err)
	}
	return item, nil
}

func (s *Items) Create(ctx context.Context, item domain.Item) (domain.Item, error) {
	if _, err := s.collection.InsertOne(ctx, item); err != nil {
		return domain.Item{}, fmt.Errorf("create MongoDB item: %w", err)
	}
	return item, nil
}

func (s *Items) Update(ctx context.Context, item domain.Item) (domain.Item, error) {
	result, err := s.collection.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: item.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: item.Name},
			{Key: "description", Value: item.Description},
			{Key: "updated_at", Value: item.UpdatedAt},
		}}},
	)
	if err != nil {
		return domain.Item{}, fmt.Errorf("update MongoDB item: %w", err)
	}
	if result.MatchedCount == 0 {
		return domain.Item{}, domain.ErrNotFound
	}
	return s.Get(ctx, item.ID)
}

func (s *Items) Delete(ctx context.Context, id string) error {
	result, err := s.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return fmt.Errorf("delete MongoDB item: %w", err)
	}
	if result.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

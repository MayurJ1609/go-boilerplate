package dependencies

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Clients struct {
	Postgres *pgxpool.Pool
	Mongo    *mongo.Client
	Redis    *redis.Client
	Kafka    *kafka.Writer

	mongoDatabase string
	kafkaBrokers  []string
}

func Connect(ctx context.Context, config Config) (*Clients, error) {
	clients := &Clients{mongoDatabase: config.MongoDatabase, kafkaBrokers: config.KafkaBrokers}

	pool, err := pgxpool.New(ctx, config.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	clients.Postgres = pool
	if err := pool.Ping(ctx); err != nil {
		return nil, clients.fail(fmt.Errorf("connect to PostgreSQL: %w", err))
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		return nil, clients.fail(fmt.Errorf("create MongoDB client: %w", err))
	}
	clients.Mongo = mongoClient
	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		return nil, clients.fail(fmt.Errorf("connect to MongoDB: %w", err))
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})
	clients.Redis = redisClient
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, clients.fail(fmt.Errorf("connect to Redis: %w", err))
	}

	dialer := &kafka.Dialer{Timeout: 5 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", config.KafkaBrokers[0])
	if err != nil {
		return nil, clients.fail(fmt.Errorf("connect to Kafka broker %s: %w", config.KafkaBrokers[0], err))
	}
	if err := connection.Close(); err != nil {
		return nil, clients.fail(fmt.Errorf("close Kafka verification connection: %w", err))
	}
	clients.Kafka = &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokers...),
		Topic:        config.KafkaTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}

	return clients, nil
}

func (c *Clients) MongoDatabase() *mongo.Database {
	return c.Mongo.Database(c.mongoDatabase)
}

func (c *Clients) Publish(ctx context.Context, key, value []byte) error {
	return c.Kafka.WriteMessages(ctx, kafka.Message{Key: key, Value: value})
}

func (c *Clients) GetCache(ctx context.Context, key string) (string, bool, error) {
	value, err := c.Redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (c *Clients) SetCache(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.Redis.Set(ctx, key, value, ttl).Err()
}

func (c *Clients) Ping(ctx context.Context) error {
	var errs []error
	if err := c.Postgres.Ping(ctx); err != nil {
		errs = append(errs, fmt.Errorf("PostgreSQL: %w", err))
	}
	if err := c.Mongo.Ping(ctx, readpref.Primary()); err != nil {
		errs = append(errs, fmt.Errorf("MongoDB: %w", err))
	}
	if err := c.Redis.Ping(ctx).Err(); err != nil {
		errs = append(errs, fmt.Errorf("Redis: %w", err))
	}
	dialer := &kafka.Dialer{Timeout: 5 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", c.kafkaBrokers[0])
	if err != nil {
		errs = append(errs, fmt.Errorf("Kafka: %w", err))
	} else if err := connection.Close(); err != nil {
		errs = append(errs, fmt.Errorf("Kafka: close verification connection: %w", err))
	}
	return errors.Join(errs...)
}

func (c *Clients) Close(ctx context.Context) error {
	var errs []error
	if c.Kafka != nil {
		if err := c.Kafka.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close Kafka writer: %w", err))
		}
	}
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close Redis: %w", err))
		}
	}
	if c.Mongo != nil {
		if err := c.Mongo.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("disconnect MongoDB: %w", err))
		}
	}
	if c.Postgres != nil {
		c.Postgres.Close()
	}
	return errors.Join(errs...)
}

func (c *Clients) fail(cause error) error {
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return errors.Join(cause, c.Close(closeCtx))
}

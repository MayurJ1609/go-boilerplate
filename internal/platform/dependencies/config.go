package dependencies

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	PostgresDSN     string
	MongoURI        string
	MongoDatabase   string
	MongoUsername   string
	MongoPassword   string
	MongoAuthSource string
	RedisAddress    string
	RedisPassword   string
	RedisDB         int
	KafkaBrokers    []string
	KafkaTopic      string
}

func LoadConfig() (Config, error) {
	redisDB, err := strconv.Atoi(envOrDefault("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("parse REDIS_DB: %w", err)
	}
	if redisDB < 0 {
		return Config{}, fmt.Errorf("REDIS_DB must be non-negative")
	}
	mongoUsername := os.Getenv("MONGODB_USERNAME")
	mongoPassword := os.Getenv("MONGODB_PASSWORD")
	if (mongoUsername == "") != (mongoPassword == "") {
		return Config{}, fmt.Errorf("MONGODB_USERNAME and MONGODB_PASSWORD must both be set or both be empty")
	}

	brokers := splitCSV(envOrDefault("KAFKA_BROKERS", "127.0.0.1:9092"))
	if len(brokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}

	return Config{
		PostgresDSN:     envOrDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/boilerplate?sslmode=disable"),
		MongoURI:        envOrDefault("MONGODB_URI", "mongodb://127.0.0.1:27017"),
		MongoDatabase:   envOrDefault("MONGODB_DATABASE", "boilerplate"),
		MongoUsername:   mongoUsername,
		MongoPassword:   mongoPassword,
		MongoAuthSource: envOrDefault("MONGODB_AUTH_SOURCE", "admin"),
		RedisAddress:    envOrDefault("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		RedisDB:         redisDB,
		KafkaBrokers:    brokers,
		KafkaTopic:      envOrDefault("KAFKA_TOPIC", "boilerplate.events"),
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if broker := strings.TrimSpace(part); broker != "" {
			result = append(result, broker)
		}
	}
	return result
}

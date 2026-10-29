package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("REDIS_DB", "")
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGODB_DATABASE", "")
	t.Setenv("MONGODB_USERNAME", "")
	t.Setenv("MONGODB_PASSWORD", "")
	t.Setenv("MONGODB_AUTH_SOURCE", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("KAFKA_TOPIC", "")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.PostgresDSN == "" || config.MongoURI == "" || config.MongoDatabase == "" {
		t.Fatal("database connection defaults must not be empty")
	}
	if config.MongoAuthSource != "admin" {
		t.Fatalf("MongoDB auth source = %q, want admin", config.MongoAuthSource)
	}
	if config.RedisAddress != "127.0.0.1:6379" || config.RedisDB != 0 {
		t.Fatalf("unexpected Redis defaults: address=%q db=%d", config.RedisAddress, config.RedisDB)
	}
	if len(config.KafkaBrokers) != 1 || config.KafkaBrokers[0] != "127.0.0.1:9092" {
		t.Fatalf("unexpected Kafka brokers: %v", config.KafkaBrokers)
	}
}

func TestLoadConfigMongoCredentials(t *testing.T) {
	t.Setenv("MONGODB_USERNAME", "app-user")
	t.Setenv("MONGODB_PASSWORD", "secret")
	t.Setenv("MONGODB_AUTH_SOURCE", "boilerplate")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.MongoUsername != "app-user" || config.MongoPassword != "secret" || config.MongoAuthSource != "boilerplate" {
		t.Fatalf("MongoDB auth config was not loaded correctly")
	}
}

func TestLoadConfigRejectsIncompleteMongoCredentials(t *testing.T) {
	t.Setenv("MONGODB_USERNAME", "app-user")
	t.Setenv("MONGODB_PASSWORD", "")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() accepted incomplete MongoDB credentials")
	}
}

func TestLoadConfigKafkaBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", " broker1:9092, ,broker2:9092 ")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(config.KafkaBrokers) != 2 || config.KafkaBrokers[0] != "broker1:9092" || config.KafkaBrokers[1] != "broker2:9092" {
		t.Fatalf("Kafka brokers = %v, want trimmed broker list", config.KafkaBrokers)
	}
}

func TestLoadConfigRejectsInvalidRedisDB(t *testing.T) {
	t.Setenv("REDIS_DB", "invalid")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() accepted invalid REDIS_DB")
	}
}

func TestLoadConfigReadsEnvExample(t *testing.T) {
	tempDir := t.TempDir()
	envFile := "POSTGRES_DSN=postgres://file-value\nREDIS_ADDR=redis-from-file:6379\n"
	if err := os.WriteFile(filepath.Join(tempDir, ".env.example"), []byte(envFile), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	t.Setenv("POSTGRES_DSN", "postgres://environment-value")
	t.Setenv("REDIS_ADDR", "")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.PostgresDSN != "postgres://environment-value" {
		t.Fatalf("Postgres DSN = %q, want environment value", config.PostgresDSN)
	}
	if config.RedisAddress != "redis-from-file:6379" {
		t.Fatalf("Redis address = %q, want value from .env.example", config.RedisAddress)
	}
}

func TestLoadConfigRejectsMalformedEnvExample(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, ".env.example"), []byte("not-a-key-value-pair\n"), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig() accepted malformed .env.example")
	}
}

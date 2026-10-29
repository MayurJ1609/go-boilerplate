# Go service boilerplate

A Go starter project with independent REST API, gRPC server, gRPC client, and cron-job entrypoints. It includes PostgreSQL, MongoDB, Redis, and Kafka clients for the REST API, gRPC server, and cron runner.

## Project structure

```text
internal/
├── config/       # Environment-based configuration
├── database/     # PostgreSQL, MongoDB, Redis, Kafka connections and migrations
├── domain/       # Core models, business logic, and repository interfaces
├── handler/      # REST and gRPC transport handlers
├── middleware/   # HTTP request logging
└── repository/   # PostgreSQL and MongoDB persistence implementations
```

Commands under `cmd/` compose these layers into each executable.

## Requirements

- Go 1.23+
- Docker (optional)

## Start dependencies

The REST API, gRPC server, and cron runner verify all four backends at startup. Start the local dependencies before running those entrypoints:

```sh
make infra-up
```

This starts PostgreSQL (`localhost:5432`), MongoDB (`localhost:27017`), Redis (`localhost:6379`), and Kafka (`127.0.0.1:9092`) using [`docker-compose.yml`](docker-compose.yml). Stop them with `make infra-down`.

If Kafka was already running when you changed its advertised listener, recreate its container with `docker compose up -d --force-recreate kafka` before restarting the app.

## Run locally

Use the Makefile targets to run an entrypoint:

```sh
make run-rest
make run-grpc-server
make run-grpc-client
make run-cron
```

`LoadConfig` reads `.env.example` from the current working directory when present. Non-empty process environment variables take precedence; if the file is absent, built-in defaults are used.

## Debug in VS Code

Install the Go extension for VS Code and Delve (`dlv`). To install Delve using the workspace, run **Terminal → Run Task → Install Delve debugger** (or run `go install github.com/go-delve/delve/cmd/dlv@latest`). Ensure Go's binary directory (`$(go env GOBIN)` or, when unset, `$(go env GOPATH)/bin`) is on VS Code's `PATH`; restart VS Code after installing if the debugger is still not found.

Then use **Terminal → Run Task → Start local dependencies**. Once the dependencies are healthy, open **Run and Debug** and choose **Debug REST API**, **Debug gRPC server**, **Debug gRPC client**, or **Debug cron**. Set breakpoints as usual. The configurations load the local development values from [`.env.example`](.env.example); change those values if your services use different addresses.

The gRPC client requires the gRPC server to be running separately. Stop the local backend containers with **Terminal → Run Task → Stop local dependencies** when finished. The **Run all tests** task runs both unit and integration test suites.

The REST API listens on `:8080`; `GET /readyz` checks the backend connections. The gRPC server listens on `:9090`; run it before starting the gRPC client.

```sh
curl 'http://localhost:8080/v1/greet?name=Ada'
```

REST routes:

- `GET /healthz` — liveness check
- `GET /readyz` — readiness check for PostgreSQL, MongoDB, Redis, and Kafka
- `GET /v1/greet?name=...` — greeting (name is optional; defaults to `World`)

The greeting endpoint caches responses in Redis for five minutes and publishes a JSON greeting event to the configured Kafka topic. On startup, the service requests topic creation through the configured Kafka bootstrap broker with one partition and one replica if needed, then waits for partition metadata to become available. The Kafka user must have permission to create topics. PostgreSQL and MongoDB clients are initialized and exposed by the shared dependency package for application persistence.

## Sample item CRUD

The REST API exposes independent item stores; operations on one backend do not write to the other:

- `GET|POST /v1/postgres/items` and `GET|PUT|DELETE /v1/postgres/items/{id}`
- `GET|POST /v1/mongo/items` and `GET|PUT|DELETE /v1/mongo/items/{id}`

Create and update requests require a non-empty `name`; `description` is optional:

```sh
curl -X POST http://localhost:8080/v1/postgres/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"example","description":"stored in PostgreSQL"}'

curl -X POST http://localhost:8080/v1/mongo/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"example","description":"stored in MongoDB"}'
```

The PostgreSQL table is created by the versioned SQL migration [`0001_create_items.sql`](internal/database/migrations/sql/0001_create_items.sql), applied automatically at service startup and recorded in `schema_migrations`. MongoDB uses the `items` collection in `MONGODB_DATABASE`, created during startup if it does not exist. New PostgreSQL migrations should be added as lexically ordered SQL files under `internal/database/migrations/sql/`.

## Configuration

| Variable | Default | Used by |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | REST server |
| `GRPC_ADDR` | `:9090` | gRPC server |
| `GRPC_TARGET` | `localhost:9090` | gRPC client |
| `CRON_SPEC` | `@every 1m` | cron runner |
| `POSTGRES_DSN` | `postgres://boilerplate:boilerplate@localhost:5432/boilerplate?sslmode=disable` | PostgreSQL |
| `MONGODB_URI` | `mongodb://127.0.0.1:27017` | MongoDB |
| `MONGODB_DATABASE` | `boilerplate` | MongoDB |
| `MONGODB_USERNAME` | empty | MongoDB (optional; set with password) |
| `MONGODB_PASSWORD` | empty | MongoDB (optional; set with username) |
| `MONGODB_AUTH_SOURCE` | `admin` | MongoDB authentication database |
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis |
| `REDIS_PASSWORD` | empty | Redis |
| `REDIS_DB` | `0` | Redis |
| `KAFKA_BROKERS` | `127.0.0.1:9092` | Kafka (comma-separated broker list) |
| `KAFKA_TOPIC` | `boilerplate.events` | Kafka producer |

See [`.env.example`](.env.example) for local development values. Configuration is in [`internal/config`](internal/config); connection lifecycle and migrations are in [`internal/database`](internal/database). Database-specific CRUD implementations are in [`internal/repository`](internal/repository), and the domain defines their shared interfaces.

For an authenticated MongoDB server, set both `MONGODB_USERNAME` and `MONGODB_PASSWORD`, and set `MONGODB_AUTH_SOURCE` to the database where that user was created (commonly `admin`). The application user also needs permission to create and use the `items` collection in `MONGODB_DATABASE`. Credentials can alternatively be embedded in `MONGODB_URI`; do not set both URI credentials and the separate username/password variables.

## Tests and build

```sh
go test ./...                         # unit tests
go test -tags=integration ./tests/integration/...
make test                             # both test suites
make build                            # build all entrypoints into bin/
make build-rest                       # build only REST into bin/rest
make build-grpc-server                # build only gRPC server into bin/grpc-server
make build-grpc-client                # build only gRPC client into bin/grpc-client
make build-cron                       # build only cron runner into bin/cron
```

## Docker

Choose the executable at build time with `APP` (`rest`, `grpc-server`, `grpc-client`, or `cron`):

```sh
docker build --build-arg APP=rest -t go-boilerplate:rest .
docker run --rm -p 8080:8080 go-boilerplate:rest
```

For the gRPC client, set `GRPC_TARGET` to the reachable server address. The image defaults to the REST executable.

## gRPC API

The service contract is in [`api/proto/greeter.proto`](api/proto/greeter.proto). Its request and response use `google.protobuf.StringValue` so the starter can keep generated code out of the repository while retaining standard protobuf wire messages.

# Go service boilerplate

A Go starter project with independent REST API, gRPC server, gRPC client, and cron-job entrypoints. It includes PostgreSQL, MongoDB, Redis, and Kafka clients for the REST API, gRPC server, and cron runner.

## Requirements

- Go 1.23+
- Docker (optional)

## Start dependencies

The REST API, gRPC server, and cron runner verify all four backends at startup. Start the local dependencies before running those entrypoints:

```sh
make infra-up
```

This starts PostgreSQL (`localhost:5432`), MongoDB (`localhost:27017`), Redis (`localhost:6379`), and Kafka (`localhost:9092`) using [`docker-compose.yml`](docker-compose.yml). Stop them with `make infra-down`.

## Run locally

Use the Makefile targets to run an entrypoint:

```sh
make run-rest
make run-grpc-server
make run-grpc-client
make run-cron
```

The REST API listens on `:8080`; `GET /readyz` checks the backend connections. The gRPC server listens on `:9090`; run it before starting the gRPC client.

```sh
curl 'http://localhost:8080/v1/greet?name=Ada'
```

REST routes:

- `GET /healthz` — liveness check
- `GET /readyz` — readiness check for PostgreSQL, MongoDB, Redis, and Kafka
- `GET /v1/greet?name=...` — greeting (name is optional; defaults to `World`)

The greeting endpoint caches responses in Redis for five minutes and publishes a JSON greeting event to the configured Kafka topic. PostgreSQL and MongoDB clients are initialized and exposed by the shared dependency package for application persistence.

## Configuration

| Variable | Default | Used by |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | REST server |
| `GRPC_ADDR` | `:9090` | gRPC server |
| `GRPC_TARGET` | `localhost:9090` | gRPC client |
| `CRON_SPEC` | `@every 1m` | cron runner |
| `POSTGRES_DSN` | `postgres://boilerplate:boilerplate@localhost:5432/boilerplate?sslmode=disable` | PostgreSQL |
| `MONGODB_URI` | `mongodb://localhost:27017` | MongoDB |
| `MONGODB_DATABASE` | `boilerplate` | MongoDB |
| `REDIS_ADDR` | `localhost:6379` | Redis |
| `REDIS_PASSWORD` | empty | Redis |
| `REDIS_DB` | `0` | Redis |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka (comma-separated broker list) |
| `KAFKA_TOPIC` | `boilerplate.events` | Kafka producer |

See [`.env.example`](.env.example) for local development values. The shared client setup is in [`internal/platform/dependencies`](internal/platform/dependencies): it opens and pings PostgreSQL, MongoDB, and Redis, verifies Kafka connectivity, provides Redis cache and Kafka publish helpers, and closes the clients on shutdown.

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

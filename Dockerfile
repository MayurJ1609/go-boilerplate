FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go-boilerplate/go.mod go-boilerplate/go.sum ./go-boilerplate/
COPY common-lib/ ./common-lib/
WORKDIR /src/go-boilerplate
RUN go mod download
COPY go-boilerplate/ ./

ARG APP=rest
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${APP}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
EXPOSE 8080 9090
ENTRYPOINT ["/app"]

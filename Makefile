BINARY  := phoenix-logs-rmq
CMD     := ./cmd/phoenix-logs-rmq
IMAGE   := rjyspl/phoenix-logs-rmq
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help
.PHONY: help run build test cover vet fmt lint tidy check docker-build docker-up docker-down clean

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | awk -F': ' '{ printf "  %-14s %s\n", $$1, substr($$0, length($$1) + 3) }'

## run: run the worker locally (reads ./.env)
run:
	go run $(CMD)

## build: compile a static binary into ./bin
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) $(CMD)

## test: run the test suite with the race detector
test:
	go test ./... -race -count=1

## cover: run tests and write an HTML coverage report to coverage.html
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

## vet: run go vet
vet:
	go vet ./...

## fmt: format all source files
fmt:
	gofmt -l -w .

## lint: run golangci-lint (https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

## tidy: sync go.mod/go.sum with imports
tidy:
	go mod tidy

## check: everything CI runs (fmt, vet, test, build)
check:
	@test -z "$$(gofmt -l .)" || (echo "run 'make fmt'"; gofmt -l .; exit 1)
	$(MAKE) vet test build

## docker-build: build the image tagged with VERSION
docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

## docker-up: start RabbitMQ, ClickHouse and the worker via docker compose
docker-up:
	docker compose up --build

## docker-down: stop the compose stack and remove its volumes
docker-down:
	docker compose down -v

## clean: remove build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

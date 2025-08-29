# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Makefile for building and deployment
.PHONY: build run test clean docker deploy

VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse HEAD)

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.Commit=$(COMMIT)

all: build discover

# Build main program
build:
	CGO_ENABLED=1 $(GOCMD) build -ldflags "$(LDFLAGS)" -o bin/raven ./cmd/raven

# Build the discovery utility
discover:
	CGO_ENABLED=1 $(GOCMD) build -ldflags "$(LDFLAGS)" -o bin/raven-discover ./cmd/raven-discover


run: build
	./bin/raven -config config.yaml

test:
	go test -v ./...

clean: clean-discover
	rm -rf bin/ data/

docker:
	docker build -t raven:$(VERSION) .
	docker tag raven:$(VERSION) raven:latest

deploy: docker
	docker-compose up -d

dev:
	go run -ldflags "$(LDFLAGS)" ./cmd/raven -config config.yaml

install-deps:

# Clean up discovery binary
clean-discover:
	rm -f bin/raven-discover

format:
	go fmt ./...
	go vet ./...

lint:
	golangci-lint run

migrate:
	@echo "Running database migrations..."
	./bin/raven -config config.yaml -migrate

backup:
	@echo "Creating database backup..."
	cp data/raven.db data/raven-backup-$(shell date +%Y%m%d-%H%M%S).db

# Development setup
setup-dev:
	@echo "Setting up development environment..."
	mkdir -p data plugins
	go mod download
	@echo "Development environment ready!"

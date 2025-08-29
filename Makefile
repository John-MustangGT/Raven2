# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Deb build info
VERSION ?= 2.0.0
ARCH ?= amd64
PACKAGE_NAME = raven
BUILD_DIR = build
DEB_DIR = $(BUILD_DIR)/$(PACKAGE_NAME)_$(VERSION)_$(ARCH)


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

clean: clean-discover clean-deb
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

# Build Debian package
.PHONY: deb
deb: build discover
	@echo "Building Debian package v$(VERSION)..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(DEB_DIR)/DEBIAN
	@mkdir -p $(DEB_DIR)/usr/bin
	@mkdir -p $(DEB_DIR)/usr/lib/raven
	@mkdir -p $(DEB_DIR)/etc/raven
	@mkdir -p $(DEB_DIR)/etc/systemd/system
	@mkdir -p $(DEB_DIR)/var/lib/raven/data
	@mkdir -p $(DEB_DIR)/var/log/raven
	@mkdir -p $(DEB_DIR)/usr/share/doc/raven

	# Copy binaries
	@cp bin/raven $(DEB_DIR)/usr/bin/
	@cp bin/raven-discover $(DEB_DIR)/usr/bin/
	@chmod 755 $(DEB_DIR)/usr/bin/raven
	@chmod 755 $(DEB_DIR)/usr/bin/raven-discover

	# Copy web assets
	@cp -r web $(DEB_DIR)/usr/lib/raven/

	# Copy Debian control files
	@cp debian/DEBIAN/control $(DEB_DIR)/DEBIAN/
	@cp debian/DEBIAN/postinst $(DEB_DIR)/DEBIAN/
	@cp debian/DEBIAN/prerm $(DEB_DIR)/DEBIAN/
	@cp debian/DEBIAN/postrm $(DEB_DIR)/DEBIAN/
	@chmod 755 $(DEB_DIR)/DEBIAN/postinst
	@chmod 755 $(DEB_DIR)/DEBIAN/prerm
	@chmod 755 $(DEB_DIR)/DEBIAN/postrm

	# Copy systemd service
	@cp debian/etc/systemd/system/raven.service $(DEB_DIR)/etc/systemd/system/

	# Create default config
	@cp debian/etc/raven/config.yaml $(DEB_DIR)/etc/raven/

	# Create documentation
	@echo "Raven Network Monitoring System v$(VERSION)" > $(DEB_DIR)/usr/share/doc/raven/README
	@echo "" >> $(DEB_DIR)/usr/share/doc/raven/README
	@echo "See /etc/raven/config.yaml for configuration options." >> $(DEB_DIR)/usr/share/doc/raven/README
	@echo "Use 'systemctl start raven' to start the service." >> $(DEB_DIR)/usr/share/doc/raven/README

	# Build the package
	@dpkg-deb --build $(DEB_DIR)
	@echo "Debian package created: $(DEB_DIR).deb"

# Install the built deb package
.PHONY: install-deb
install-deb: deb
	@echo "Installing Raven Debian package..."
	@sudo dpkg -i $(DEB_DIR).deb

# Clean up build artifacts
.PHONY: clean-deb
clean-deb:
	@rm -rf $(BUILD_DIR)

# Create directory structure for development
.PHONY: setup-debian
setup-debian:
	@mkdir -p debian/DEBIAN
	@mkdir -p debian/etc/raven
	@mkdir -p debian/etc/systemd/system


# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Deb build info
RELEASE_VERSION ?= 2.0.0
ARCH ?= amd64
PACKAGE_NAME = raven
BUILD_DIR = build
DEB_DIR = $(BUILD_DIR)/$(PACKAGE_NAME)_$(RELEASE_VERSION)_$(ARCH)


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
	docker build -t raven:$(RELEASE_VERSION) .
	docker tag raven:$(RELEASE_VERSION) raven:latest

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
	@echo "Building Debian package v$(RELEASE_VERSION)..."
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
	@cp config/raven.service $(DEB_DIR)/etc/systemd/system/

	# Create default config
	@cp debian/etc/raven/config.yaml $(DEB_DIR)/etc/raven/

	# Copy documentation files
	@mkdir -p $(DEB_DIR)/usr/share/doc/raven/examples
	@cp README.md $(DEB_DIR)/usr/share/doc/raven/
	@cp docs/Configuration.md $(DEB_DIR)/usr/share/doc/raven/
	@cp config/alert-rules.yaml $(DEB_DIR)/usr/share/doc/raven/examples/
	@cp config/prometheus.yml $(DEB_DIR)/usr/share/doc/raven/examples/
	
	# Create installation summary
	@echo "Raven Network Monitoring System v$(RELEASE_VERSION)" > $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "=========================================" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "Installation Summary:" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Configuration:     /etc/raven/config.yaml" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Data Directory:    /var/lib/raven/" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Web Interface:     http://localhost:8000" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Service Control:   systemctl start/stop/status raven" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Logs:              journalctl -u raven -f" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "Quick Start:" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  1. sudo raven-discover -network 192.168.1.0/24 -output /tmp/config.yaml" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  2. sudo cp /tmp/config.yaml /etc/raven/config.yaml" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  3. sudo systemctl start raven" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "Documentation:" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Main README:       /usr/share/doc/raven/README.md" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Configuration:     /usr/share/doc/raven/Configuration.md" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL
	@echo "  Examples:          /usr/share/doc/raven/examples/" >> $(DEB_DIR)/usr/share/doc/raven/INSTALL

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


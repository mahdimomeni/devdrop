# DevDrop Makefile

BINARY_NAME=devdrop
SERVER_SRC=./cmd/server
WEB_DIR=./web

.PHONY: all build-web build-server run clean test

all: build-web build-server

# Build Svelte SPA assets
build-web:
	@echo "==> Building Svelte SPA..."
	cd $(WEB_DIR) && npm install && npm run build

# Build self-contained Go server with embedded assets
build-server:
	@echo "==> Compiling Go server binary..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) $(SERVER_SRC)

# Run DevDrop server on default port 8080
run:
	@echo "==> Starting DevDrop LAN Server..."
	go run $(SERVER_SRC)

# Clean build artifacts
clean:
	@echo "==> Cleaning artifacts..."
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf $(WEB_DIR)/dist data/

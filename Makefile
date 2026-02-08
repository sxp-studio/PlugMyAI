BINARY_NAME=plug-my-ai
DAEMON_DIR=daemon
UI_DIR=daemon-ui/dashboard
TOOLBAR_DIR=daemon-ui/toolbar/macOS
EMBED_DIR=$(DAEMON_DIR)/internal/dashboard/dist
BIN_DIR=bin

.PHONY: all build dashboard daemon clean dev macos-app

all: build

# Build everything: dashboard then daemon
build: dashboard daemon

# Build the Svelte dashboard and copy to embed directory
dashboard:
	cd $(UI_DIR) && npm install && npm run build
	rm -rf $(EMBED_DIR)
	cp -r $(UI_DIR)/dist $(EMBED_DIR)

# Build the Go daemon binary
daemon:
	cd $(DAEMON_DIR) && go build -o ../$(BIN_DIR)/$(BINARY_NAME) ./cmd/plug-my-ai

# Build daemon without rebuilding dashboard (uses existing embedded assets)
daemon-only:
	cd $(DAEMON_DIR) && go build -o ../$(BIN_DIR)/$(BINARY_NAME) ./cmd/plug-my-ai

# Development: run daemon with --no-tray flag
dev: build
	./$(BIN_DIR)/$(BINARY_NAME) --no-tray

# Cross-compilation targets
build-darwin-arm64:
	cd $(DAEMON_DIR) && GOOS=darwin GOARCH=arm64 go build -o ../$(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/plug-my-ai

build-darwin-amd64:
	cd $(DAEMON_DIR) && GOOS=darwin GOARCH=amd64 go build -o ../$(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/plug-my-ai

build-linux-amd64:
	cd $(DAEMON_DIR) && GOOS=linux GOARCH=amd64 go build -o ../$(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/plug-my-ai

build-linux-arm64:
	cd $(DAEMON_DIR) && GOOS=linux GOARCH=arm64 go build -o ../$(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/plug-my-ai

build-all: dashboard build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-linux-arm64

# Build macOS menu bar app (dashboard + daemon universal binary + Swift app)
macos-app: dashboard
	cd $(TOOLBAR_DIR) && $(MAKE) bundle

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(UI_DIR)/dist
	rm -rf $(UI_DIR)/node_modules
	cd $(TOOLBAR_DIR) && $(MAKE) clean

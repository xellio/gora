BINARY_NAME=gora
SETUP_NAME=gora-setup
BUILD_DIR=bin
DOCKER_COMPOSE=docker-compose

.PHONY: all build setup run clean help up down restart status

all: build 

up:
	@echo "Starting Redis Stack..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Redis Insight is available at http://localhost:8001"

down:
	@echo "Stopping Redis..."
	@$(DOCKER_COMPOSE) down

restart: down up
	@echo "Redis restarted"

status:
	@$(DOCKER_COMPOSE) ps

build:
	@echo "Building CLI..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/prompt/main.go
	@go build -o $(BUILD_DIR)/$(SETUP_NAME) ./cmd/database/setup.go
	@echo "Done! Binaries are in $(BUILD_DIR)/"

setup:
	@echo "Populating Database..."
	@go run ./cmd/database/setup.go

wipe:
	@echo "Wiping Redis database..."
	@docker exec -it $$(docker ps -qf "name=redis") redis-cli FLUSHALL
	@echo "Redis is now empty and clean."

run:
	@echo "Starting GoRa CLI..."
	@go run ./cmd/prompt/main.go

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned!"

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Infrastructure:"
	@echo "  up       - Start Redis Stack (Docker)"
	@echo "  down     - Stop Redis"
	@echo "  wipe     - Wipe Redis database"
	@echo "  status   - Show Docker status"
	@echo ""
	@echo "Application:"
	@echo "  build    - Compile all binaries"
	@echo "  setup    - Run DB population"
	@echo "  run      - Start GoRa CLI"
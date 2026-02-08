BINARY_NAME=gora
SETUP_NAME=gora-setup
BUILD_DIR=bin
DOCKER_COMPOSE=docker-compose
BACKUP_DIR=backups
REDIS_CONTAINER=$(shell docker ps -qf "name=redis")
REDIS_SERVICE=redis-vector
DATE=$(shell date +%Y-%m-%d_%H%M)

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

backup:
	@mkdir -p $(BACKUP_DIR)
	@echo "Creating Redis snapshot..."
	@docker exec -it $(REDIS_CONTAINER) redis-cli SAVE
	@echo "Saving to $(BACKUP_DIR)/dump_$(DATE).rdb..."
	@docker cp $(REDIS_CONTAINER):/data/dump.rdb $(BACKUP_DIR)/dump_$(DATE).rdb
	@cp $(BACKUP_DIR)/dump_$(DATE).rdb $(BACKUP_DIR)/dump_latest.rdb
	@echo "Backup complete: dump_$(DATE).rdb"

restore:
	@echo "Restoring from $(BACKUP_DIR)/dump_latest.rdb..."
	@if [ ! -f $(BACKUP_DIR)/dump_latest.rdb ]; then echo "Error: No dump_latest.rdb found!"; exit 1; fi
	@$(DOCKER_COMPOSE) stop $(REDIS_SERVICE)
	@docker cp $(BACKUP_DIR)/dump_latest.rdb $(REDIS_CONTAINER):/data/dump.rdb
	@$(DOCKER_COMPOSE) start $(REDIS_SERVICE)
	@echo "Waiting for Redis to load data..."
	@sleep 2
	@echo "Restore complete!"

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
.PHONY: all build run test clean docker-build docker-up docker-down docker-logs help deps lint

# Variables
APP_NAME=multi-worker
MAIN_PATH=./cmd/server
DOCKER_COMPOSE=docker compose

# Default target
all: deps build

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build the application
build:
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

# Run the application locally
run:
	go run $(MAIN_PATH)

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint the code (requires golangci-lint)
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker commands
docker-build:
	docker build -t $(APP_NAME):latest .

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-up-dev:
	$(DOCKER_COMPOSE) --profile dev up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f app

docker-restart:
	$(DOCKER_COMPOSE) restart app

docker-rebuild:
	$(DOCKER_COMPOSE) up -d --build app

# Database commands
db-shell:
	$(DOCKER_COMPOSE) exec postgres psql -U postgres -d multiworker

db-reset:
	$(DOCKER_COMPOSE) down -v
	$(DOCKER_COMPOSE) up -d postgres
	sleep 5
	$(DOCKER_COMPOSE) up -d app

# Generate API documentation (requires swag)
docs:
	swag init -g cmd/server/main.go -o docs

# Create a sample task via API
sample-task:
	@echo "Creating sample job scraper task..."
	@curl -X POST http://localhost:8080/api/v1/tasks \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer YOUR_TOKEN_HERE" \
		-d '{ \
			"name": "Golang Remote Jobs", \
			"description": "Scrape golang remote jobs and notify Discord", \
			"schedule": "0 */2 * * *", \
			"pipeline": [ \
				{ \
					"type": "scraper", \
					"config": { \
						"source": "remoteok", \
						"query": "golang", \
						"limit": 10 \
					} \
				}, \
				{ \
					"type": "ai_processor", \
					"config": { \
						"provider": "openai", \
						"prompt": "Summarize these job listings. Highlight salary, requirements, and remote status." \
					} \
				}, \
				{ \
					"type": "discord", \
					"config": {} \
				} \
			] \
		}'

# Help
help:
	@echo "Available targets:"
	@echo "  deps           - Download and tidy Go dependencies"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application locally"
	@echo "  dev            - Run with hot reload (requires air)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Lint the code"
	@echo "  clean          - Remove build artifacts"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-up      - Start all containers"
	@echo "  docker-up-dev  - Start all containers with dev profile"
	@echo "  docker-down    - Stop all containers"
	@echo "  docker-logs    - Follow app container logs"
	@echo "  docker-rebuild - Rebuild and restart app container"
	@echo "  db-shell       - Open PostgreSQL shell"
	@echo "  db-reset       - Reset database (WARNING: deletes all data)"
	@echo "  sample-task    - Create a sample task via API"

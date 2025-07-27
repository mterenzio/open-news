# Open News Makefile

.PHONY: build run test clean deps migrate dev seed test-basic

# Build the application
build:
	go build -o bin/open-news cmd/main.go

# Run the application
run:
	go run cmd/main.go

# Seed the database with initial data
seed:
	go run cmd/seed.go

# Test basic functionality (no database required)
test-basic:
	go run cmd/test.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Setup environment
setup-env:
	cp .env.example .env
	@echo "Environment file created! Edit .env with your database settings."

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run database migrations (requires running PostgreSQL)
migrate:
	go run cmd/main.go migrate

# Development mode with live reload (requires air)
dev:
	air

# Install air for live reload
install-air:
	go install github.com/cosmtrek/air@latest

# Docker commands
docker-build:
	docker build -t open-news .

docker-run:
	docker compose up

docker-dev:
	docker compose up postgres -d
	@echo "Waiting for PostgreSQL to start..."
	@sleep 10
	@echo "PostgreSQL is ready! Run 'make run' in another terminal"

docker-stop:
	docker compose down

# Database setup (PostgreSQL with Homebrew)
db-install:
	brew install postgresql@15
	brew services start postgresql@15

db-setup:
	createdb open_news
	@echo "Database 'open_news' created successfully!"
	@echo "Make sure to copy .env.example to .env and configure it"

db-drop:
	dropdb open_news

db-start:
	brew services start postgresql@15

db-stop:
	brew services stop postgresql@15

db-status:
	brew services list | grep postgresql

# Lint the code
lint:
	golangci-lint run

# Format the code
fmt:
	go fmt ./...

# Testing commands
test-api:
	@echo "Testing API endpoints..."
	curl -s http://localhost:8080/health && echo
	curl -s http://localhost:8080/api/worker/status | jq
	curl -s http://localhost:8080/api/feeds/global | jq

test-feeds:
	@echo "Testing AT Protocol feed endpoints..."
	curl -s "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-global" | jq
	curl -s -H "Authorization: Bearer test-token" "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-personal" | jq

test-db:
	@echo "Testing database connection..."
	psql -l | grep open_news
	@echo "Database exists!"

# Help
help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  restart      - Restart the application server"
	@echo "  stop         - Stop the application server"
	@echo "  status       - Show server status"
	@echo "  logs         - Show server logs"
	@echo "  seed         - Seed the database"
	@echo "  test         - Run Go tests"
	@echo "  test-api     - Test API endpoints"
	@echo "  test-feeds   - Test feed endpoints"
	@echo "  test-db      - Test database connection"
	@echo "  migrate      - Run database migrations"
	@echo "  clean        - Clean build artifacts"
	@echo "  setup-env    - Create .env file"
	@echo "  db-setup     - Create database"
	@echo "  fmt          - Format code"
	@echo ""
	@echo "Visit http://localhost:8080 for the developer dashboard"

# Development server management
restart:
	@echo "Restarting server..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@sleep 1
	@make run &
	@sleep 3
	@echo "Server restarted on http://localhost:8080"

stop:
	@echo "Stopping server..."
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@echo "Server stopped"

status:
	@echo "Server status:"
	@if lsof -ti:8080 > /dev/null 2>&1; then \
		echo "✅ Server is running on port 8080"; \
		curl -s http://localhost:8080/health || echo "❌ Health check failed"; \
	else \
		echo "❌ Server is not running"; \
	fi

logs:
	@if [ -f server.log ]; then \
		tail -f server.log; \
	else \
		echo "No log file found. Run 'make run > server.log 2>&1 &' to start with logging"; \
	fi

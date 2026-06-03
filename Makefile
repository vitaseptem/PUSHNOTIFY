.PHONY: dev build up down logs migrate seed test sdk dashboard server fmt

# Bring the full stack up in dev mode.
dev:
	docker compose up --build

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

# Production build of all images.
build:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml build

# Run database migrations inside the server image.
migrate:
	docker compose run --rm server /app/pushnotify migrate

# Populate the database with example data.
seed:
	docker compose run --rm server /app/pushnotify seed

# Go unit tests.
test:
	cd server && go test ./...

# Format Go code.
fmt:
	cd server && gofmt -w .

# Build the publishable TypeScript SDK.
sdk:
	cd sdk && npm install && npm run build

# Build the dashboard.
dashboard:
	cd dashboard && npm install && npm run build

# Build the Go server binary locally.
server:
	cd server && go build -o bin/pushnotify ./cmd/pushnotify

.PHONY: help dev backend frontend docker-up docker-down lint test build fmt migrate

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

## ── Backend ──────────────────────────────────────────────

backend: ## Run backend in development mode
	cd backend && DB_PASSWORD=$${DB_PASSWORD:-change_me_in_production} go run ./cmd/api

fmt: ## Format backend code
	cd backend && go fmt ./...

vet: ## Run go vet
	cd backend && go vet ./...

test: ## Run backend tests
	cd backend && go test ./...

build-backend: ## Build backend binary
	cd backend && go build -o bin/api ./cmd/api

## ── Frontend ─────────────────────────────────────────────

frontend: ## Run frontend in development mode
	cd frontend && npm run dev

lint: ## Lint frontend
	cd frontend && npm run lint

build-frontend: ## Build frontend for production
	cd frontend && npm run build

## ── Docker ───────────────────────────────────────────────

docker-up: ## Start all services with Docker Compose
	cd deploy && docker compose up -d

docker-down: ## Stop all Docker services
	cd deploy && docker compose down

docker-build: ## Build Docker images
	cd deploy && docker compose build

docker-logs: ## Follow logs
	cd deploy && docker compose logs -f

docker-validate: ## Validate docker compose file
	cd deploy && docker compose config --quiet && echo "docker-compose.yml is valid"

## ── Database ─────────────────────────────────────────────

migrate-up: ## Run all pending migrations
	cd backend && migrate -path ./migrations -database "postgresql://$${DB_USER:-utp}:$${DB_PASSWORD}@$${DB_HOST:-localhost}:$${DB_PORT:-5432}/$${DB_NAME:-unified_trust}?sslmode=disable" up

migrate-down: ## Roll back the last migration
	cd backend && migrate -path ./migrations -database "postgresql://$${DB_USER:-utp}:$${DB_PASSWORD}@$${DB_HOST:-localhost}:$${DB_PORT:-5432}/$${DB_NAME:-unified_trust}?sslmode=disable" down 1

## ── CI ───────────────────────────────────────────────────

ci: fmt vet test build-backend build-frontend docker-validate ## Run all CI checks

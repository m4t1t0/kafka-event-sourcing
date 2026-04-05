.DEFAULT_GOAL := help
POSTGRES_DSN ?= postgres://postgres:postgres@localhost:5432/projections?sslmode=disable

.PHONY: help build run-client run-order dev-client dev-order infra infra-down stop clean migrate-up migrate-down migrate-create

##@ Helpers 🚀

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build all service binaries
	go build -o bin/client-service ./cmd/client-service
	go build -o bin/order-service ./cmd/order-service

up: ## Start everything (infra + services) in Docker
	docker-compose up -d --build

infra: ## Start infrastructure only (Kafka, Zookeeper, PostgreSQL)
	docker-compose up -d zookeeper kafka kafka-init ksqldb kafka-ui postgres

infra-down: ## Stop infrastructure
	docker-compose down

run-client: ## Run client-service locally
	go run ./cmd/client-service

run-order: ## Run order-service locally
	go run ./cmd/order-service

dev-client: ## Run client-service with hot reload (air)
	air -c .air.client.toml

dev-order: ## Run order-service with hot reload (air)
	air -c .air.order.toml

stop: ## Stop infrastructure and remove volumes
	docker-compose down -v

##@ Database 💾

.PHONY: db-create
db-create: ## Create the database, if not exists
	docker compose exec postgres psql -U postgres -d postgres -c "CREATE DATABASE projections;"

.PHONY: db-drop
db-drop: ## Drop the database
	docker compose exec postgres psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS projections;"

migrate-up: ## Run all pending migrations
	migrate -path migrations -database "$(POSTGRES_DSN)" up

migrate-down: ## Roll back the last migration
	migrate -path migrations -database "$(POSTGRES_DSN)" down 1

clean: ## Remove build artifacts
	rm -rf bin/ tmp/

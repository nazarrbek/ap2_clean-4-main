.PHONY: up down build tidy logs

## Start all services with Docker Compose
up:
	docker compose up --build -d

## Stop all services
down:
	docker compose down -v

## Build all images without starting
build:
	docker compose build

## Download/update dependencies for all services
tidy:
	cd order-service && go mod tidy
	cd payment-service && go mod tidy
	cd notification-service && go mod tidy

## Show logs for all services
logs:
	docker compose logs -f

## Show logs for a specific service: make logs-s s=order-service
logs-s:
	docker compose logs -f $(s)

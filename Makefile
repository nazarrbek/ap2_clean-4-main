.PHONY: tidy build up down

## Download dependencies for both services
tidy:
	cd payment-contract && go mod tidy
	cd order-contract && go mod tidy
	cd order-service && go mod tidy
	cd payment-service && go mod tidy

## Build both services locally
build:
	cd order-service && go build ./cmd/order-service
	cd payment-service && go build ./cmd/payment-service

## Start everything with Docker Compose
up:
	docker-compose up --build

## Stop everything
down:
	docker-compose down -v

## Run locally (requires PostgreSQL running)
run-payment:
	cd payment-service && GRPC_PORT=50051 go run ./cmd/payment-service

run-order:
	cd order-service && PAYMENT_GRPC_ADDR=localhost:50051 GRPC_PORT=50052 go run ./cmd/order-service

## Apply migrations manually
migrate-orders:
	psql -U postgres -d orders_db -f order-service/migrations/001_init.sql

migrate-payments:
	psql -U postgres -d payments_db -f payment-service/migrations/001_init.sql

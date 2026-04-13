# AP2 Assignment 2 - gRPC Migration & Contract-First Development

## What was migrated

This project now uses gRPC for internal service-to-service communication:

- Order Service remains a REST API for end users (`POST /orders`, etc.).
- Order Service calls Payment Service through gRPC (`ProcessPayment`).
- Payment Service runs a gRPC server and returns typed protobuf responses.
- Order Service also runs a gRPC server with server-side streaming:
  - `SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate)`

Business logic in domain/usecase layers remains unchanged; migration is done in transport/composition layers.

## Contract-First setup

### Repository A (proto contracts)
- https://github.com/nazarrbek/ap2-proto-contracts

### Repository B (generated code)
- https://github.com/nazarrbek/ap2-generated-contracts

In this workspace, generated contracts are represented as shared modules:

- `order-contract/`
- `payment-contract/`

Proto sources:

- `proto/order.proto`
- `proto/payment.proto`

## gRPC contracts

### Payment Service
- Service: `payment.PaymentService`
- RPC: `ProcessPayment(PaymentRequest) returns (PaymentResponse)`
- Uses `int64` for money and `google.protobuf.Timestamp` for `processed_at`

### Order Service Streaming
- Service: `order.OrderService`
- RPC: `SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate)`
- Stream sends:
  - initial current status
  - every real status change detected from DB-backed order reads

## High-level architecture

```text
Client
  -> REST (Gin)
Order Service :8080
  -> gRPC Client (PaymentService.ProcessPayment)
Payment Service :50051

Streaming client/frontend
  -> gRPC SubscribeToOrderUpdates
Order Service :50052
  -> polls real order state from repository/usecase and pushes only status changes
```

## Environment variables

### Order Service
- `DB_HOST` (default: `localhost`)
- `DB_PORT` (default: `5432`)
- `DB_USER` (default: `postgres`)
- `DB_PASSWORD` (default: `postgres`)
- `DB_NAME` (default: `orders_db`)
- `SERVER_PORT` (default: `8080`) - REST
- `GRPC_PORT` (default: `50052`) - order streaming gRPC server
- `PAYMENT_GRPC_ADDR` (default: `localhost:50051`) - payment gRPC endpoint

### Payment Service
- `DB_HOST` (default: `localhost`)
- `DB_PORT` (default: `5432`)
- `DB_USER` (default: `postgres`)
- `DB_PASSWORD` (default: `postgres`)
- `DB_NAME` (default: `payments_db`)
- `SERVER_PORT` (default: `8081`) - REST
- `GRPC_PORT` (default: `50051`) - payment gRPC server

## Run with Docker Compose

```bash
docker compose up --build
```

Ports:
- Order REST: `8080`
- Payment REST: `8081`
- Payment gRPC: `50051`
- Order gRPC streaming: `50052`

## Run locally

```bash
# Terminal 1
cd payment-service
GRPC_PORT=50051 go run ./cmd/payment-service

# Terminal 2
cd order-service
PAYMENT_GRPC_ADDR=localhost:50051 GRPC_PORT=50052 go run ./cmd/order-service
```

## Protobuf generation (local fallback)

```bash
protoc \
  --plugin=protoc-gen-go=/Users/nazarbek/go/bin/protoc-gen-go \
  --plugin=protoc-gen-go-grpc=/Users/nazarbek/go/bin/protoc-gen-go-grpc \
  --go_out=. \
  --go-grpc_out=. \
  proto/payment.proto proto/order.proto
```

## Test: REST order creation with internal gRPC payment call

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c-1","item_name":"Laptop","amount":50000}'
```

Expected behavior:
- REST request succeeds on Order Service.
- Order Service calls Payment Service over gRPC.
- Final order status becomes `Paid` or `Failed` (for declined cases).

## Test: server-side streaming with real DB-backed updates

1) Create order and copy `id`:

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c-2","item_name":"Book","amount":900}'
```

2) Start stream subscriber:

```bash
cd order-service
go run ./cmd/order-stream-client --addr localhost:50052 --order-id <ORDER_ID>
```

3) Trigger status change through business API (updates DB):

```bash
curl -X PATCH http://localhost:8080/orders/<ORDER_ID>/cancel
```

Expected streaming result:
- First message: current order status.
- Next message appears immediately after DB status change (e.g., `Cancelled`).

## Bonus: gRPC interceptor

Payment Service includes a unary interceptor that logs:
- full gRPC method name
- request duration
- error value

Example log:

```text
grpc method=/payment.PaymentService/ProcessPayment duration=1.8ms err=<nil>
```

## Deliverables checklist

- Updated Order and Payment services with gRPC migration
- README with contract repo links and run instructions
- Updated architecture section (REST external + gRPC internal)
- Evidence to attach in report/LMS:
  - successful gRPC payment call (via order creation)
  - successful streaming updates on real status change

## Extra files for defense

- Architecture diagram: docs/ARCHITECTURE.md
- Execution evidence from real run: docs/EVIDENCE.md
- Defense checklist and script: docs/DEFENSE.md


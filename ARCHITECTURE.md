# AP2 Assignment 4 — Architecture

## Services & Ports

| Service | Port | Description |
|---------|------|-------------|
| order-service | 8080 | Order management API + Redis cache + rate limiter |
| payment-service | 8081 | Payment processing API + RabbitMQ publisher |
| notification-service | — | Background worker (no HTTP port) |
| redis | 6379 | Cache + rate limiter + idempotency store |
| rabbitmq | 5672 / 15672 | Message broker (Management UI at :15672) |
| orders-db | 5432 | PostgreSQL for orders |
| payments-db | 5433 | PostgreSQL for payments |

## Data Flow

1. `POST /orders` → Order created in PostgreSQL, cached in Redis (5 min TTL)
2. `GET /orders/{id}` → Cache-aside: Redis HIT returns cached, MISS queries DB then caches
3. `POST /payments` → Payment saved to DB, PaymentEvent published to RabbitMQ
4. Notification Service (background worker) receives event:
   - Idempotency check: Redis `GET notif:idempotency:{payment_id}`
   - If "done" → skip (no duplicate email)
   - `SET NX` to mark as processing (atomic)
   - Call `EmailSender.Send()` via Adapter (SIMULATED or SMTP)
   - Retry with exponential backoff: 2s → 4s → 8s on failure
   - On success: `SET` key to "done", ACK message
   - On total failure: DEL key, NACK (RabbitMQ requeues)
5. `PATCH /orders/{id}/status` → DB update + Redis cache DELETED (atomic invalidation)

## Key Design Decisions

### Cache-Aside Pattern (Order Service)
- Before DB query, check Redis
- On miss: DB → Redis (populate)
- On status change: Redis key DELETED immediately (not after TTL)
- This prevents "Pending" being shown for a paid order

### Adapter Pattern (Notification Service)
- `EmailSender` interface: `Send(ctx, to, subject, body) error`
- `SimulatedEmailSender`: logs, adds latency, random failures
- `SMTPEmailSender`: real SMTP
- Selection via `PROVIDER_MODE=SIMULATED|REAL` env var
- Use case has NO knowledge of concrete providers

### Idempotency (Notification Service)
- `SET NX notif:idempotency:{payment_id}` is atomic
- Prevents duplicate emails even with concurrent workers
- Failed jobs: key deleted → next delivery can retry

### Rate Limiter (Bonus, Order Service)
- Redis `INCR` + `EXPIRE` per client IP
- Shared across all replicas
- Returns HTTP 429 + `X-RateLimit-*` headers when exceeded

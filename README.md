# AP2 Assignment 4 — Performance Optimization & External Integrations

**Student:** Taubakabyl Nurlybek  
**Course:** Advanced Programming 2

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AP2 Assignment 4 System                              │
├──────────────────┬───────────────────────┬──────────────────────────────────┤
│   Order Service  │   Payment Service     │   Notification Service           │
│   :8080          │   :8081               │   (background worker)            │
│                  │                       │                                  │
│ ┌──────────────┐ │ ┌──────────────────┐  │ ┌──────────────────────────────┐ │
│ │  HTTP API    │ │ │    HTTP API      │  │ │   RabbitMQ Consumer          │ │
│ │  + Rate      │ │ │                  │  │ │   (Background Worker)        │ │
│ │  Limiter     │ │ │  ProcessPayment  │──┼▶│                              │ │
│ └──────┬───────┘ │ │  → publishes to  │  │ │ ┌──────────────────────────┐ │ │
│        │         │ │    RabbitMQ      │  │ │ │  Idempotency Check       │ │ │
│ ┌──────▼───────┐ │ └──────────────────┘  │ │ │  (Redis SET NX)          │ │ │
│ │  Use Case    │ │                       │ │ └────────────┬─────────────┘ │ │
│ │  (cache-     │ │ ┌──────────────────┐  │ │              │               │ │
│ │   aside)     │ │ │  PostgreSQL      │  │ │ ┌────────────▼─────────────┐ │ │
│ └──────┬───────┘ │ │  (payments DB)   │  │ │ │  EmailSender Interface   │ │ │
│        │         │ └──────────────────┘  │ │ │  (Adapter Pattern)       │ │ │
│ ┌──────▼───────┐ │                       │ │ │  SIMULATED | REAL SMTP   │ │ │
│ │  Redis Cache │ │                       │ │ └────────────┬─────────────┘ │ │
│ │  (5 min TTL) │◀┼───── invalidation ────┼─┤              │               │ │
│ └──────┬───────┘ │                       │ │ ┌────────────▼─────────────┐ │ │
│        │         │                       │ │ │  Retry + Exp. Backoff    │ │ │
│ ┌──────▼───────┐ │                       │ │ │  (2s → 4s → 8s)          │ │ │
│ │  PostgreSQL  │ │                       │ │ └──────────────────────────┘ │ │
│ │  (orders DB) │ │                       │ └──────────────────────────────┘ │
│ └──────────────┘ │                       │                                  │
└──────────────────┴───────────────────────┴──────────────────────────────────┘
                                    │
              ┌─────────────────────┼───────────────────┐
              │                     │                   │
          ┌───▼───┐            ┌────▼────┐        ┌────▼────┐
          │ Redis │            │RabbitMQ │        │Postgres │
          │ :6379 │            │  :5672  │        │ x2 DBs  │
          └───────┘            └─────────┘        └─────────┘
```

---

## Assignment 4 Features

### 1. Redis Caching — Cache-Aside Pattern (Order Service)

**Strategy:** Cache-Aside (Lazy Loading)

**Read path:**
1. Check Redis for `order:{id}` key
2. **Cache HIT** → return immediately (no DB query)
3. **Cache MISS** → query PostgreSQL → store result in Redis with 5-min TTL → return

**Invalidation Strategy:**
- When `PATCH /orders/{id}/status` is called (e.g., after payment), the use case:
  1. Updates the database
  2. **Immediately deletes** the Redis key — atomic invalidation
  3. Returns fresh data from DB on the next read

This ensures stale data (e.g., "pending" for a paid order) is never served.

**Code location:** `order-service/internal/cache/order_cache.go`, `order-service/internal/usecase/order_usecase.go`

### 2. Adapter Pattern — EmailSender (Notification Service)

The `EmailSender` interface in `notification-service/internal/adapter/email_sender.go` decouples business logic from providers:

```go
type EmailSender interface {
    Send(ctx context.Context, to, subject, body string) error
}
```

**Providers:**
- `SimulatedEmailSender` — logs the action, adds latency (`time.Sleep`), has a configurable random failure rate to test retry logic
- `SMTPEmailSender` — real SMTP integration

**Selection:** Set `PROVIDER_MODE=SIMULATED` or `PROVIDER_MODE=REAL` in `.env`

### 3. Reliable Background Jobs (Notification Service)

The notification service is a **background worker** that:

**Idempotency (Redis SET NX):**
```
Before sending → SET NX notif:idempotency:{payment_id} "processing"
After success  → SET notif:idempotency:{payment_id} "done"  
On failure     → DEL notif:idempotency:{payment_id}  (allows retry)
```

**Retry with Exponential Backoff:**
- Attempt 1: wait 2s
- Attempt 2: wait 4s
- Attempt 3: wait 8s
- After max retries: NACK message → RabbitMQ requeues

**Code location:** `notification-service/internal/usecase/notification_usecase.go`

### Bonus: Redis Rate Limiter (+10%)

The Order Service applies a rate limiter middleware that uses Redis `INCR + EXPIRE`:

- **Limit:** 10 requests/minute per IP (configurable via `RATE_LIMIT` env var)
- **Response headers:** `X-RateLimit-Limit`, `X-RateLimit-Remaining`
- **HTTP 429** when limit exceeded

**Code location:** `order-service/internal/middleware/rate_limiter.go`

---

## Quick Start

```bash
# 1. Clone/enter the project
cd ass4

# 2. Start everything
docker compose up --build -d

# 3. Check logs
docker compose logs -f notification-service

# 4. Stop
docker compose down -v
```

## API Reference

### Order Service (port 8080)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/orders` | Create order |
| `GET` | `/orders` | List recent orders |
| `GET` | `/orders/{id}` | Get order (cache-aside) |
| `PATCH` | `/orders/{id}/status` | Update status (+ cache invalidation) |

### Payment Service (port 8081)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/payments` | Process payment (triggers notification job) |
| `GET` | `/payments/{id}` | Get payment |
| `GET` | `/payments/order/{orderID}` | Get payment by order |

## Demo Flow

```bash
# 1. Create an order
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-001","item_name":"Laptop","amount":999.99}' | jq

# 2. Get the order (cache MISS → DB query, stored in Redis)
curl -s http://localhost:8080/orders/1 | jq

# 3. Get again (cache HIT — check order-service logs for "cache HIT")
curl -s http://localhost:8080/orders/1 | jq

# 4. Process payment (publishes event to RabbitMQ)
curl -s -X POST http://localhost:8081/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":1,"customer_email":"john@example.com","amount":999.99}' | jq

# Watch notification-service logs — retry logic and idempotency in action:
docker compose logs -f notification-service

# 5. Update order status (invalidates cache)
curl -s -X PATCH http://localhost:8080/orders/1/status \
  -H "Content-Type: application/json" \
  -d '{"status":"paid","payment_id":"pay-001"}' | jq
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT` | `10` | Requests per minute per IP |
| `CACHE_TTL` | `300` | Order cache TTL in seconds |
| `PROVIDER_MODE` | `SIMULATED` | Email provider: `SIMULATED` or `REAL` |
| `SIMULATE_FAIL_RATE` | `0.3` | 30% random failure in simulated mode |
| `SIMULATE_LATENCY_MS` | `200` | Latency in ms for simulated mode |
| `MAX_RETRIES` | `3` | Max retry attempts with exponential backoff |
| `IDEMPOTENCY_TTL_HOURS` | `24` | How long to keep idempotency records |

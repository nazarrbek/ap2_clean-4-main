# Architecture – Assignment 4

## System Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                        CLIENT (HTTP)                                 │
└─────────────────────────────┬────────────────────────────────────────┘
                              │ POST /orders  GET /orders/:id
                              ▼
┌──────────────────────────────────────────────────────────────────────┐
│                     ORDER SERVICE (:8080)                            │
│                                                                      │
│  ┌─────────────────┐    ┌──────────────────────────────────────┐    │
│  │ Rate Limiter    │    │  HTTP Handler                        │    │
│  │ Middleware      │───►│  (gin)                               │    │
│  │ [Redis INCR]    │    └──────────────┬───────────────────────┘    │
│  └─────────────────┘                   │                            │
│                                        ▼                            │
│                          ┌─────────────────────────┐               │
│                          │    OrderUseCase          │               │
│                          │                          │               │
│                          │  GetOrder():             │               │
│                          │  1. cache.Get(id)  ──────┼──► Redis      │
│                          │     HIT → return         │   (Cache)     │
│                          │     MISS → continue      │               │
│                          │  2. repo.GetByID(id) ────┼──► PostgreSQL │
│                          │  3. cache.Set(order)─────┼──► Redis      │
│                          │                          │               │
│                          │  CreateOrder():          │               │
│                          │  1. repo.Create()   ─────┼──► PostgreSQL │
│                          │  2. paymentClient   ─────┼──► gRPC       │
│                          │  3. repo.UpdateStatus────┼──► PostgreSQL │
│                          │  4. cache.Delete() ──────┼──► Redis ✗    │
│                          │     (INVALIDATE)         │               │
│                          └────────────┬────────────┘               │
└───────────────────────────────────────┼────────────────────────────┘
                                        │ gRPC
                                        ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    PAYMENT SERVICE (:8081 / :50051)                  │
│                                                                      │
│   PaymentUseCase → PostgreSQL (payments_db)                          │
│   Publishes PaymentEvent → RabbitMQ (payments exchange)              │
└──────────────────────────────────────────────────────────────────────┘
                                        │ AMQP publish
                                        ▼
                               ┌─────────────────┐
                               │    RabbitMQ      │
                               │  payments exch.  │
                               │  → payment.      │
                               │    completed q.  │
                               │  → payments.dlx  │
                               │    (on failure)  │
                               └────────┬─────────┘
                                        │ AMQP consume
                                        ▼
┌──────────────────────────────────────────────────────────────────────┐
│                  NOTIFICATION SERVICE (Background Worker)            │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ RabbitMQ Consumer                                           │    │
│  │  handleDelivery(msg)                                        │    │
│  │       │                                                     │    │
│  │       ▼                                                     │    │
│  │  NotificationUseCase.Handle(event)                          │    │
│  │       │                                                     │    │
│  │  1. IdempotencyStore.IsProcessed(payment_id)                │    │
│  │       │  HIT  ───► return false, nil (skip duplicate)       │    │
│  │       │  MISS ───► continue                                 │    │
│  │       │                                                     │    │
│  │  2. retry.Do(maxAttempts=5, base=2s):                       │    │
│  │       │  EmailSender.Send(msg)   ◄── Adapter Pattern        │    │
│  │       │    │                                                │    │
│  │       │    ├── SIMULATED: random 30% failure + latency      │    │
│  │       │    └── REAL:      SMTP / Mailjet                    │    │
│  │       │                                                     │    │
│  │       │  fail → wait 2s → retry                             │    │
│  │       │  fail → wait 4s → retry                             │    │
│  │       │  fail → wait 8s → retry                             │    │
│  │       │  fail → wait 16s → retry                            │    │
│  │       │  fail → wait 30s → retry                            │    │
│  │       │  all fail → NACK → Dead Letter Queue                │    │
│  │       │                                                     │    │
│  │  3. IdempotencyStore.MarkProcessed(payment_id)              │    │
│  │       │  Redis SET notification:idempotency:{id} TTL=24h    │    │
│  │       │                                                     │    │
│  │  4. ACK message                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                │                                    │
│                         Redis (Idempotency)                         │
└──────────────────────────────────────────────────────────────────────┘

## Infrastructure
┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────┐
│ orders-db   │  │ payments-db │  │ Redis :6379                 │
│ (PostgreSQL)│  │ (PostgreSQL)│  │  • order:{id}   (cache)     │
│             │  │             │  │  • ratelimit:{ip} (counter) │
└─────────────┘  └─────────────┘  │  • notif:idempotency:{id}  │
                                   └─────────────────────────────┘
```

## Cache Invalidation Flow

```
POST /orders  or  PATCH /orders/:id/cancel
        │
        ├─ 1. Write to PostgreSQL   (source of truth)
        ├─ 2. cache.Delete(order.id) → Redis DEL order:{id}
        └─ 3. Return HTTP response

Next GET /orders/:id:
        └─ cache miss → fresh read from PostgreSQL → cache repopulated
```

## Retry Backoff Schedule

| Attempt | Delay before | Cumulative wait |
|---------|-------------|-----------------|
| 1 | 0s | 0s |
| 2 | 2s | 2s |
| 3 | 4s | 6s |
| 4 | 8s | 14s |
| 5 | 16s (capped 30s) | 30s |
| → DLQ | — | — |

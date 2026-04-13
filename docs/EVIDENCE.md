# Evidence Log (2026-04-13)

This file captures successful runtime evidence for Assignment 2 requirements.

## 1) Stack startup

Command:

```bash
docker compose up --build
```

Result:
- order-service: running on 8080 (REST) and 50052 (gRPC stream)
- payment-service: running on 8081 (REST) and 50051 (gRPC)
- orders-db and payments-db: healthy

## 2) REST order creation with internal gRPC call

Command:

```bash
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c-step1","item_name":"Phone","amount":900}'
```

Response:

```json
{"amount":900,"created_at":"2026-04-13T17:49:31.636615212Z","customer_id":"c-step1","id":"ae5bfa9b-aec8-4225-b052-a740e19fbe45","item_name":"Phone","status":"Paid"}
```

This confirms external REST works and order business flow finishes with payment result.

## 3) Payment gRPC interceptor log

Command:

```bash
docker logs ap2_clean-4-main-payment-service-1 --tail 80
```

Observed log line:

```text
grpc method=/payment.PaymentService/ProcessPayment duration=5.898833ms err=<nil>
```

This confirms internal call Order -> Payment is gRPC and interceptor is active.

## 4) Server-side streaming tied to DB change

Subscriber command:

```bash
cd order-service
go run ./cmd/order-stream-client --addr localhost:50052 --order-id ae5bfa9b-aec8-4225-b052-a740e19fbe45
```

Initial stream output:

```text
2026/04/13 22:49:50 order=ae5bfa9b-aec8-4225-b052-a740e19fbe45 status=Paid emitted_at=2026-04-13T17:49:50Z
```

DB status update command:

```bash
docker exec ap2_clean-4-main-orders-db-1 \
  psql -U postgres -d orders_db \
  -c "UPDATE orders SET status='Cancelled' WHERE id='ae5bfa9b-aec8-4225-b052-a740e19fbe45';"
```

DB update result:

```text
UPDATE 1
```

Next stream output:

```text
2026/04/13 22:50:18 order=ae5bfa9b-aec8-4225-b052-a740e19fbe45 status=Cancelled emitted_at=2026-04-13T17:50:18Z
```

This demonstrates real-time streaming from real database state changes (not fake timer-only simulation).

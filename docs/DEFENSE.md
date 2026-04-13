# Defense Guide (2-3 minutes)

## 1) Fast demo flow

1. Start stack:
   - docker compose up --build
2. Confirm containers:
   - docker ps
3. Create order via REST (Order Service):
   - curl -s -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d '{"customer_id":"demo-user","item_name":"Phone","amount":900}'
4. Show Payment interceptor log (proves internal gRPC call):
   - docker logs ap2_clean-4-main-payment-service-1 --tail 80
5. Start stream subscriber (Order gRPC stream):
   - cd order-service
   - go run ./cmd/order-stream-client --addr localhost:50052 --order-id <ORDER_ID>
6. Trigger real DB status update:
   - docker exec ap2_clean-4-main-orders-db-1 psql -U postgres -d orders_db -c "UPDATE orders SET status='Cancelled' WHERE id='<ORDER_ID>';"
7. Show stream receives new status immediately.

## 2) What to say during defense

- External API remains REST in Order Service (Gin).
- Internal Order -> Payment communication migrated to gRPC.
- Contract-first approach is used via proto contracts and generated shared code modules.
- Use case and domain logic are preserved; migration is in transport/composition layers.
- Streaming endpoint is server-side gRPC from Order Service.
- Streaming output is tied to real DB state changes, not fake timer-only simulation.
- Payment Service has unary interceptor logging method, duration, and error.

## 3) Required screenshots checklist

1. docker compose up --build successful startup.
2. docker ps with all 4 containers up and DB healthy.
3. POST /orders response with created order id and status.
4. Payment Service log line with:
   - grpc method=/payment.PaymentService/ProcessPayment
5. Stream client first output (initial status).
6. SQL UPDATE result (UPDATE 1).
7. Stream client second output after DB change (status changed).
8. README section with links to Proto Repo and Generated Repo (after you insert real links).
9. Architecture diagram file view in docs/ARCHITECTURE.md.

## 4) Common questions and short answers

Q: Why keep REST if migration is to gRPC?
A: Assignment requires REST for end-user entrypoints; only internal service communication is migrated to gRPC.

Q: How is Clean Architecture preserved?
A: Domain and usecase layers are unchanged; adapters in transport layer were switched to gRPC.

Q: How do you prove stream is real?
A: After changing status directly in orders DB, stream emits new status immediately.

Q: Where are status codes handled?
A: In gRPC handlers using google.golang.org/grpc/status and grpc/codes.

Q: Where is interceptor implemented?
A: In payment-service/internal/app/app.go via grpc.UnaryInterceptor.

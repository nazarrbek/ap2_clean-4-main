# Architecture Diagram (Assignment 2)

```mermaid
flowchart LR
    C[Client / Frontend] -->|HTTP REST| OHTTP[Order Service :8080\nGin Handlers]

    OHTTP --> OUC[Order Use Case]
    OUC --> OREP[(orders_db)]

    OUC -->|gRPC ProcessPayment| PGRPC[Payment Service :50051\ngRPC Server]
    PGRPC --> PUC[Payment Use Case]
    PUC --> PREP[(payments_db)]

    SC[Streaming Client] -->|gRPC SubscribeToOrderUpdates| OGRPC[Order Service :50052\ngRPC Streaming Server]
    OGRPC --> OUC
```

## Notes

- External API remains REST via Gin in Order Service.
- Internal communication Order -> Payment is gRPC.
- Streaming endpoint belongs to Order Service and emits updates from real DB state changes.

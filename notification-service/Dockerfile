FROM golang:1.24-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /app/notification-service
COPY notification-service .
RUN GOSUMDB=off go mod graph > /dev/null 2>&1 || true && \
    go build -mod=mod -o /app/notification-service-bin ./cmd/notification-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/notification-service-bin ./notification-service
CMD ["./notification-service"]

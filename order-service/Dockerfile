FROM golang:1.24-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY order-contract ./order-contract
COPY payment-contract ./payment-contract
COPY order-service ./order-service
WORKDIR /app/order-service
RUN go mod download
RUN go build -o /app/order-service-bin ./cmd/order-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service-bin ./order-service
EXPOSE 8080
CMD ["./order-service"]

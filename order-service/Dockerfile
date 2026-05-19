FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY order-contract ./order-contract
COPY payment-contract ./payment-contract
COPY order-service/go.mod ./order-service/
WORKDIR /app/order-service
RUN go mod tidy && go mod download
COPY order-service ./
RUN go build -o /app/order-service-bin ./cmd/order-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service-bin ./order-service
EXPOSE 8080
CMD ["./order-service"]

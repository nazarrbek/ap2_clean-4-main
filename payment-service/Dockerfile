FROM golang:1.24-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY payment-contract ./payment-contract
COPY payment-service ./payment-service
WORKDIR /app/payment-service
RUN go mod download
RUN go build -o /app/payment-service-bin ./cmd/payment-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/payment-service-bin ./payment-service
EXPOSE 8081
CMD ["./payment-service"]

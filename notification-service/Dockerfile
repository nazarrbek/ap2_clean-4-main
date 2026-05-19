FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY notification-service/go.mod ./notification-service/
WORKDIR /app/notification-service
RUN go mod tidy && go mod download
COPY notification-service ./
RUN go build -o /app/notification-service-bin ./cmd/notification-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/notification-service-bin ./notification-service
CMD ["./notification-service"]

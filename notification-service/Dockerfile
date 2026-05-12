FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go mod download
RUN go build -o /app/notification-service ./cmd/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/notification-service /usr/local/bin/notification-service
CMD ["/usr/local/bin/notification-service"]

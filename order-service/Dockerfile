FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go mod download
RUN go build -o /app/order-service ./cmd/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/order-service /usr/local/bin/order-service
EXPOSE 8080
CMD ["/usr/local/bin/order-service"]

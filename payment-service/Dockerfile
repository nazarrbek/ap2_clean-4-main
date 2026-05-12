FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go mod download
RUN go build -o /app/payment-service ./cmd/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/payment-service /usr/local/bin/payment-service
EXPOSE 8081
CMD ["/usr/local/bin/payment-service"]

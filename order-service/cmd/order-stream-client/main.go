package main

import (
	"context"
	"flag"
	"io"
	"log"
	"time"

	orderpb "order-contract/order"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:50052", "order grpc address")
	orderID := flag.String("order-id", "", "order id to subscribe")
	flag.Parse()

	if *orderID == "" {
		log.Fatal("order-id is required")
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := orderpb.NewOrderServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stream, err := client.SubscribeToOrderUpdates(ctx, &orderpb.OrderRequest{OrderId: *orderID})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			log.Println("stream closed by server")
			return
		}
		if err != nil {
			log.Fatalf("recv: %v", err)
		}

		log.Printf("order=%s status=%s emitted_at=%s", update.GetOrderId(), update.GetStatus(), update.GetEmittedAt().AsTime().Format(time.RFC3339))
	}
}

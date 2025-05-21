package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	pb "try/pkg/grpcapi"

	"google.golang.org/grpc"
)

var services = []string{
	"user-service",
	"order-service",
	"payment-service",
	"inventory-service",
}

var backendCounts = make(map[string]int)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewSidecarServiceClient(conn)

	ticker := time.NewTicker(1 * time.Second)
	summaryTicker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	defer summaryTicker.Stop()

	for {
		select {
		case <-ticker.C:
			go sendRequest(client)
		case <-summaryTicker.C:
			fmt.Println("--- Backend Distribution (last 10s) ---")
			for backend, count := range backendCounts {
				fmt.Printf("%s: %d requests\n", backend, count)
			}
			// Reset counts for next interval
			backendCounts = make(map[string]int)
		}
	}
}

func sendRequest(client pb.SidecarServiceClient) {
	// Pick a random service
	service := services[rand.Intn(len(services))]

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	resp, err := client.RouteRequest(ctx, &pb.RouteRequestRequest{
		ServiceName: service,
	})

	if err != nil {
		log.Printf("Error routing %s: %v", service, err)
		return
	}

	fmt.Printf("Routed %s to %s\n", service, resp.Backend)
	backendCounts[resp.Backend]++
}

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
}

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewSidecarServiceClient(conn)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		go sendRequest(client)
	}
}

func sendRequest(client pb.SidecarServiceClient) {
	service := services[rand.Intn(len(services))]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.RouteRequest(ctx, &pb.RouteRequestRequest{
		ServiceName: service,
	})

	if err != nil {
		log.Printf("Error routing %s: %v", service, err)
		return
	}

	fmt.Printf("Routed %s to %s\n", service, resp.Backend)
}

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	pb "try/pkg/grpcapi"

	"google.golang.org/grpc"
)

var services = []string{
	"user-service",
}
var portToService = map[string]string{
	"35476": "user-service-a",
	"35188": "user-service-b",
	"35067": "user-service-c",
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

// func sendRequest(client pb.SidecarServiceClient) {
// 	service := services[rand.Intn(len(services))]
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	resp, err := client.RouteRequest(ctx, &pb.RouteRequestRequest{
// 		ServiceName: service,
// 	})

// 	if err != nil {
// 		log.Printf("Error routing %s: %v", service, err)
// 		return
// 	}

//		fmt.Printf("Routed %s to %s\n", service, resp.Backend)
//	}
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

	// Extract port from resp.Backend (e.g., "http://69.157.137.154:35067")
	parts := strings.Split(resp.Backend, ":")
	port := parts[len(parts)-1]

	serviceName, ok := portToService[port]
	if !ok {
		serviceName = "unknown-service"
	}

	fmt.Printf("Routed %s to %s (port: %s)\n", service, serviceName, port)
}

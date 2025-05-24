package main

import (
	"log"
	"net"

	pb "try/pkg/grpcapi"
	"try/pkg/server"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSidecarServiceServer(grpcServer, &server.SidecarServer{})

	log.Println("Sidecar gRPC server is running on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

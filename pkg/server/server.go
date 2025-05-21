package server

import (
	"log"
	"net"

	pb "try/pkg/grpcapi"

	"google.golang.org/grpc"
)

func Start() error {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSidecarServiceServer(grpcServer, &SidecarServer{})
	log.Println("Running LoadBalancer")
	return grpcServer.Serve(listener)
}

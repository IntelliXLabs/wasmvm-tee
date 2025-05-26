package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/IntelliXLabs/dtvm-tee/dtvm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Define command line arguments
	var port = flag.Int("port", 50051, "gRPC server port")
	flag.Parse()

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", *port, err)
	}

	// Create gRPC server
	s := grpc.NewServer()

	// Register DTVM TEE service
	server := &dtvm.Server{}
	dtvm.RegisterDTVMTeeServiceServer(s, server)

	// Enable reflection (optional, for debugging and tooling support)
	reflection.Register(s)

	log.Printf("DTVM TEE gRPC server starting on port %d...", *port)
	log.Printf("Server listening at %v", lis.Addr())

	// Start the server
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

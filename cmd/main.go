package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	pb "bloombox/api/bloom_pb"
	"bloombox/internal/server"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. Port for grpc
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = ":50051"
	}
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 2. create a gRPC server
	grpcEngine := grpc.NewServer()

	// 3. init handler
	bloomLogic := &server.BloomServer{}

	// 4. register handler with the enginer
	pb.RegisterBloomServiceServer(grpcEngine, bloomLogic)

	// 5. gRPC server in a seperate background thread
	go func() {
		log.Printf("Bloombox gRPC server listing at port %v", lis.Addr())
		if err := grpcEngine.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// 6. set up rest proxy (grpc-gateway)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	// proxy connect to grpc server
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = pb.RegisterBloomServiceHandlerFromEndpoint(ctx, mux, "localhost"+grpcPort, opts)

	if err != nil {
		log.Fatalf("Failed to start HTTP gateway: %v", err)
	}

	// 7. Start HTTP server
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = ":8080"
	}
	log.Printf("Bloombox rest gateway listening at port %v", httpPort)
	if err := http.ListenAndServe(httpPort, mux); err != nil {
		log.Fatalf("Failed to server HTTP: %v", err)
	}

}

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"doric/backend/internal/auth"
	"doric/backend/internal/db"
	"doric/backend/internal/deploy"
	pb "doric/backend/pkg/proto/doric"
)

type server struct {
	pb.UnimplementedDoricServiceServer
	orchestrator *deploy.Orchestrator
}

func main() {
	log.Println("--- Doric Control Plane Starting ---")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	ctx := context.Background()

	// Initialize Database
	userRepo, err := db.NewUserRepo(ctx)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	if err := userRepo.InitDB(ctx); err != nil {
		log.Fatalf("failed to initialize db schema: %v", err)
	}

	// Initialize Services
	orchestrator := deploy.NewOrchestrator()
	authSvc := auth.NewAuthService()
	authHandler := auth.NewGrpcHandler(authSvc, userRepo)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	
	// Register Services
	pb.RegisterDoricServiceServer(s, &server{orchestrator: orchestrator})
	pb.RegisterAuthServiceServer(s, authHandler)

	// Enable reflection for debugging
	reflection.Register(s)

	log.Printf("Control Plane listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Implement gRPC handlers here after code generation
// func (s *server) Build(req *pb.BuildRequest, stream pb.DoricService_BuildServer) error { ... }

package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

// AgentServer handles instructions from Doric Cloud
type AgentServer struct {
	// dockerClient *client.Client
}

// Deploy triggers a deployment on this VPS
func (s *AgentServer) Deploy(ctx context.Context, repoUrl string) error {
	log.Printf("Received deployment instruction for: %s", repoUrl)
	// 1. Git Clone
	// 2. Docker Build/Pull
	// 3. Traefik Label Configuration
	return nil
}

func main() {
	log.Println("--- Doric VPS Agent Starting ---")

	// Verify Docker Connectivity
	// cli, err := client.NewClientWithOpts(client.FromEnv)
	// if err != nil {
	// 	log.Fatalf("Error connecting to Docker: %v", err)
	// }

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// Register Agent Service
	// pb.RegisterAgentServiceServer(s, &AgentServer{})

	log.Printf("Agent listening on port 50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

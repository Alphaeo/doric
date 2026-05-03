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

// ── gRPC server ───────────────────────────────────────────────────────────

type server struct {
	pb.UnimplementedDoricServiceServer
	orchestrator *deploy.Orchestrator
}

// Deploy streame les logs de déploiement depuis l'agent vers Electron.
func (s *server) Deploy(req *pb.DeployRequest, stream pb.DoricService_DeployServer) error {
	log.Printf("[Deploy] project=%s repo=%s branch=%s env=%s",
		req.ProjectId, req.RepoUrl, req.Branch, req.TargetEnv)

	dreq := deploy.DeployRequest{
		ProjectID: req.ProjectId,
		RepoURL:   req.RepoUrl,
		Branch:    req.Branch,
		Domain:    req.Domain,
		TargetEnv: req.TargetEnv,
		EnvVars:   req.EnvVars,
	}

	err := s.orchestrator.Deploy(stream.Context(), dreq, func(line string, progress float32, done bool, success bool, appURL string) {
		status := "deploying"
		if done && success {
			status = "live"
		} else if done && !success {
			status = "failed"
		}

		_ = stream.Send(&pb.DeployResponse{
			LogLine:  line,
			Progress: progress,
			Done:     done,
			Success:  success,
			AppUrl:   appURL,
			Status:   status,
		})
	})

	if err != nil {
		log.Printf("[Deploy] error: %v", err)
		_ = stream.Send(&pb.DeployResponse{
			LogLine: "Erreur interne: " + err.Error(),
			Done:    true,
			Success: false,
			Status:  "failed",
		})
	}
	return nil
}

// Build — stub pour les builds locaux (Phase 3)
func (s *server) Build(req *pb.BuildRequest, stream pb.DoricService_BuildServer) error {
	_ = stream.Send(&pb.BuildResponse{LogLine: "Build local non encore implémenté", Success: false})
	return nil
}

// Logs — stream les logs d'un container via l'agent
func (s *server) Logs(req *pb.LogRequest, stream pb.DoricService_LogsServer) error {
	_ = stream.Send(&pb.LogResponse{
		Component: "system",
		Message:   "Log streaming via agent — bientôt disponible",
	})
	return nil
}

// Control — start/stop/restart d'une app
func (s *server) Control(ctx context.Context, req *pb.ControlRequest) (*pb.ControlResponse, error) {
	return &pb.ControlResponse{Success: false, Message: "Non implémenté"}, nil
}

// ── Main ──────────────────────────────────────────────────────────────────

func main() {
	log.Println("--- Doric Control Plane démarrage ---")

	if err := godotenv.Load(); err != nil {
		log.Println("Pas de .env, utilisation des variables système")
	}

	ctx := context.Background()

	userRepo, err := db.NewUserRepo(ctx)
	if err != nil {
		log.Fatalf("DB init error: %v", err)
	}
	if err := userRepo.InitDB(ctx); err != nil {
		log.Fatalf("DB schema error: %v", err)
	}

	orchestrator := deploy.NewOrchestrator()
	authSvc := auth.NewAuthService()
	authHandler := auth.NewGrpcHandler(authSvc, userRepo)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDoricServiceServer(s, &server{orchestrator: orchestrator})
	pb.RegisterAuthServiceServer(s, authHandler)
	reflection.Register(s)

	log.Printf("Control Plane en écoute sur %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve error: %v", err)
	}
}

// Vérifie que PORT est défini pour les healthchecks docker-compose
var _ = os.Getenv

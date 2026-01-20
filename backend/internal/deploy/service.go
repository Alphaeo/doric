package deploy

import (
	"context"
	"fmt"
)

type DeploymentRequest struct {
	ProjectID string
	RepoURL   string
	Branch    string
}

type Service interface {
	Deploy(ctx context.Context, req DeploymentRequest) error
}

type Orchestrator struct {
	// dependencies like Providers and Agents
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{}
}

func (o *Orchestrator) Deploy(ctx context.Context, req DeploymentRequest) error {
	fmt.Printf("Orchestrating deployment for project %s from %s\n", req.ProjectID, req.RepoURL)
	// 1. Validate
	// 2. Select Provider/VPS
	// 3. Send Instruction to Agent
	return nil
}

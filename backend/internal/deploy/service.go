package deploy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// agentLogLine correspond au JSON streamed par l'agent.
type agentLogLine struct {
	Log     string `json:"log"`
	Done    bool   `json:"done"`
	Success bool   `json:"success"`
	AppURL  string `json:"app_url"`
}

// DeployRequest est le contrat interne backend ↔ handlers gRPC.
type DeployRequest struct {
	ProjectID string
	RepoURL   string
	Branch    string
	Domain    string
	TargetEnv string
	EnvVars   map[string]string
}

// LogCallback est appelé pour chaque ligne de log reçue de l'agent.
type LogCallback func(line string, progress float32, done bool, success bool, appURL string)

// Orchestrator gère la sélection d'agent et la transmission des déploiements.
type Orchestrator struct {
	agentAddr string
	client    *http.Client
}

func NewOrchestrator() *Orchestrator {
	addr := os.Getenv("AGENT_ADDR")
	if addr == "" {
		addr = "http://agent:50052"
	}
	return &Orchestrator{
		agentAddr: addr,
		client: &http.Client{
			Timeout: 0, // pas de timeout global : le stream peut durer plusieurs minutes
			Transport: &http.Transport{
				ResponseHeaderTimeout: 10 * time.Second,
			},
		},
	}
}

// Deploy appelle l'agent et stream les logs via cb jusqu'à la fin du déploiement.
func (o *Orchestrator) Deploy(ctx context.Context, req DeployRequest, cb LogCallback) error {
	body, err := json.Marshal(map[string]interface{}{
		"project_id": req.ProjectID,
		"repo_url":   req.RepoURL,
		"branch":     req.Branch,
		"domain":     req.Domain,
		"env_vars":   req.EnvVars,
	})
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.agentAddr+"/deploy", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request build error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("agent unreachable (%s): %w", o.agentAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var step float32 = 0
	for scanner.Scan() {
		var line agentLogLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}

		// Estimation grossière de la progression
		switch {
		case containsAny(line.Log, "[git]"):
			step = 0.2
		case containsAny(line.Log, "[docker]", "Step "):
			step = min32(step+0.05, 0.8)
		case line.Done:
			step = 1.0
		}

		cb(line.Log, step, line.Done, line.Success, line.AppURL)

		if line.Done {
			break
		}
	}
	return scanner.Err()
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

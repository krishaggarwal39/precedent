package adapters

import (
	"context"
	"time"

	"github.com/precedent-cli/precedent/internal/types"
)

// AgentResult represents the execution outcome of an agent.
type AgentResult struct {
	CostUSD     float64
	TotalTokens int
	Duration    time.Duration
	Error       error
	Unverified  bool
}

// AgentAdapter defines the interface for interacting with different AI coding agents.
type AgentAdapter interface {
	Name() string
	Run(ctx context.Context, workDir string, taskPrompt string, task types.Task) (*AgentResult, error)
	IsInstalled() bool
}

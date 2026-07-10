package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/precedent-cli/precedent/internal/types"
)

// DummyAdapter is a mock agent used for testing the pipeline without incurring LLM costs.
type DummyAdapter struct{}

func (d *DummyAdapter) Name() string {
	return "dummy-agent"
}

func (d *DummyAdapter) IsInstalled() bool {
	return true
}

func (d *DummyAdapter) Run(ctx context.Context, workDir string, taskPrompt string, task types.Task) (*AgentResult, error) {
	fmt.Printf("[DummyAgent] Executing task %s in %s\n", task.InstanceID, workDir)
	
	// Simulate work
	select {
	case <-time.After(2 * time.Second):
		fmt.Printf("[DummyAgent] Completed task %s\n", task.InstanceID)
	case <-ctx.Done():
		fmt.Printf("[DummyAgent] Cancelled task %s\n", task.InstanceID)
		return &AgentResult{Error: ctx.Err()}, ctx.Err()
	}

	return &AgentResult{
		CostUSD:     0.0,
		TotalTokens: 42,
		Duration:    2 * time.Second,
		Error:       nil,
	}, nil
}

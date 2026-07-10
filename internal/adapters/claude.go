package adapters

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/precedent-cli/precedent/internal/types"
)

// ClaudeAdapter integrates with the anthropic claude-code CLI.
type ClaudeAdapter struct{}

func (c *ClaudeAdapter) Name() string {
	return "claude-code"
}

func (c *ClaudeAdapter) IsInstalled() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *ClaudeAdapter) Run(ctx context.Context, workDir string, taskPrompt string, task types.Task) (*AgentResult, error) {
	start := time.Now()

	// Command: claude -p "<prompt>"
	// Since we are running unattended, claude-code might need specific flags or env vars.
	// For V1, we just spawn it with the prompt.
	cmd := exec.CommandContext(ctx, "claude", "-p", taskPrompt)
	cmd.Dir = workDir

	// Create a new process group so we can reliably kill it and its children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// We don't pipe stdout/stderr to terminal because this runs concurrently
	// But we should capture it for the result.
	out, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		// Cleanly kill the process group if it timed out or failed
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return &AgentResult{
			Duration: duration,
			Error:    fmt.Errorf("claude execution failed: %v (output: %s)", err, out),
		}, err
	}

	// Parse cost from claude's output (mocked for V1 since we don't have exact stdout schema yet)
	// Example: In a real implementation, we would regex search for "Cost: $X.XX"

	return &AgentResult{
		CostUSD:     0.0, // TODO: parse from output
		TotalTokens: 0,   // TODO: parse from output
		Duration:    duration,
		Error:       nil,
	}, nil
}

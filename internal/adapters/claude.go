package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/precedent-cli/precedent/internal/types"
)

// ClaudeAdapter integrates with the anthropic claude-code CLI.
type ClaudeAdapter struct{}

func init() {
	Register("claude", func() AgentAdapter { return &ClaudeAdapter{} })
}

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
	cmd := exec.CommandContext(ctx, "claude", "-p", taskPrompt, "--output-format", "json")
	cmd.Dir = workDir

	// Create a new process group so we can reliably kill it and its children
	configureProcAttr(cmd)
	cmd.Cancel = func() error {
		return killProcessGroup(cmd)
	}
	cmd.WaitDelay = 10 * time.Second

	out, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		return &AgentResult{
			Duration: duration,
			Error:    fmt.Errorf("claude execution failed: %v (output: %s)", err, out),
		}, err
	}

	costUSD, tokens, parseErr := parseClaudeOutput(out)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Warning: could not parse cost data from claude output: %v\n", parseErr)
	}

	return &AgentResult{
		CostUSD:     costUSD,
		TotalTokens: tokens,
		Duration:    duration,
		Error:       nil,
	}, nil
}

type claudeOutput struct {
	TotalCostUSD float64 `json:"total_cost_usd"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

func parseClaudeOutput(data []byte) (float64, int, error) {
	var out claudeOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return 0, 0, err
	}
	totalTokens := out.Usage.InputTokens + out.Usage.OutputTokens
	return out.TotalCostUSD, totalTokens, nil
}

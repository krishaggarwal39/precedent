package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/precedent-cli/precedent/internal/config"
	"github.com/precedent-cli/precedent/internal/types"
)

// YamlAdapter runs an arbitrary bash command defined in agents.yaml
type YamlAdapter struct {
	AgentName string
	Config    config.AgentConfig
}

func (y *YamlAdapter) Name() string {
	return y.AgentName
}

func (y *YamlAdapter) IsInstalled() bool {
	// Custom agents are assumed to be installed if they are configured.
	return true
}

func (y *YamlAdapter) Run(ctx context.Context, workDir string, taskPrompt string, task types.Task) (*AgentResult, error) {
	start := time.Now()

	// Substitute placeholders securely in the shell string
	cmdStr := strings.ReplaceAll(y.Config.Command, "{{PROMPT}}", "\"$PRECEDENT_PROMPT\"")
	cmdStr = strings.ReplaceAll(cmdStr, "{{WORKTREE}}", "\"$PRECEDENT_WORKTREE\"")

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = workDir

	// Inject environment variables safely
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PRECEDENT_PROMPT=%s", taskPrompt))
	cmd.Env = append(cmd.Env, fmt.Sprintf("PRECEDENT_WORKTREE=%s", workDir))
	for _, e := range y.Config.Env {
		envStr := strings.ReplaceAll(e, "{{WORKTREE}}", workDir)
		envStr = strings.ReplaceAll(envStr, "{{PROMPT}}", taskPrompt)
		cmd.Env = append(cmd.Env, envStr)
	}

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
			Error:    fmt.Errorf("agent '%s' failed: %v\nOutput: %s", y.AgentName, err, out),
		}, err
	}

	return &AgentResult{
		CostUSD:     0.0, // Custom metric parsing can be added later
		TotalTokens: 0,
		Duration:    duration,
		Error:       nil,
	}, nil
}

package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
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

	// Substitute placeholders
	cmdStr := strings.ReplaceAll(y.Config.Command, "{{PROMPT}}", taskPrompt)
	cmdStr = strings.ReplaceAll(cmdStr, "{{WORKTREE}}", workDir)

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = workDir

	// Inject custom environment variables
	cmd.Env = os.Environ()
	for _, e := range y.Config.Env {
		envStr := strings.ReplaceAll(e, "{{WORKTREE}}", workDir)
		cmd.Env = append(cmd.Env, envStr)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	out, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
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

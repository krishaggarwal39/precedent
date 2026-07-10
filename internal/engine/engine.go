package engine

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/runner"
	"github.com/precedent-cli/precedent/internal/types"
)

type EventType string

const (
	EventTaskStarted    EventType = "TASK_STARTED"
	EventTaskFinished   EventType = "TASK_FINISHED"
	EventBudgetExceeded EventType = "BUDGET_EXCEEDED"
)

type Event struct {
	Type       EventType
	InstanceID string
	Result     *adapters.AgentResult
	Message    string
}

type Engine struct {
	cfg   Config
	agent adapters.AgentAdapter
}

func New(cfg Config, agent adapters.AgentAdapter) *Engine {
	return &Engine{
		cfg:   cfg,
		agent: agent,
	}
}

// Run executes the given tasks and streams events to the returned channel.
// The channel is closed when all tasks are complete.
func (e *Engine) Run(ctx context.Context, tasks []types.Task) (<-chan Event, []*adapters.AgentResult, error) {
	events := make(chan Event, len(tasks)*2)
	results := make([]*adapters.AgentResult, len(tasks))

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(e.cfg.Concurrency)

	var currentCost float64
	var costMu sync.Mutex
	var resMu sync.Mutex

	// We launch the execution loop in a background goroutine so we can return the channel immediately.
	go func() {
		defer close(events)

		for i, task := range tasks {
			t := task
			idx := i
			g.Go(func() error {
				if gCtx.Err() != nil {
					return nil // Global context cancelled
				}

				events <- Event{Type: EventTaskStarted, InstanceID: t.InstanceID}

				wtDir := filepath.Join(e.cfg.WorktreeDir, t.InstanceID)
				isolation := runner.NewGitIsolation(e.cfg.RepoPath, wtDir, t.BaseCommit)

				if err := isolation.Setup(gCtx); err != nil {
					resMu.Lock()
					results[idx] = &adapters.AgentResult{Error: fmt.Errorf("isolation setup failed: %w", err)}
					resMu.Unlock()
					events <- Event{Type: EventTaskFinished, InstanceID: t.InstanceID, Result: results[idx]}
					return nil
				}
				defer isolation.Teardown(context.Background())

				agentCtx, agentCancel := context.WithTimeout(gCtx, e.cfg.TaskTimeout)
				defer agentCancel()

				var result *adapters.AgentResult
				result, err := e.agent.Run(agentCtx, wtDir, t.ProblemStatement, t)
				if result == nil {
					result = &adapters.AgentResult{Error: err}
				}

				if result.Error == nil && e.cfg.TestCommand != "" {
					var testCmd *exec.Cmd
					if e.cfg.DockerImage != "" {
						absWtDir, _ := filepath.Abs(wtDir)
						dockerArgs := []string{"run", "--rm", "--network=none", "--memory=2g", "--cpus=2", "--cap-drop=ALL", "--security-opt=no-new-privileges", "-v", fmt.Sprintf("%s:/workspace", absWtDir), "-w", "/workspace", e.cfg.DockerImage, "sh", "-c", e.cfg.TestCommand}
						testCmd = exec.CommandContext(agentCtx, "docker", dockerArgs...)
					} else {
						testCmd = exec.CommandContext(agentCtx, "sh", "-c", e.cfg.TestCommand)
						testCmd.Dir = wtDir
					}

					out, err := testCmd.CombinedOutput()
					if err != nil {
						result.Error = fmt.Errorf("Test failed: %v\nOutput: %s", err, string(out))
					}
				} else if result.Error == nil && e.cfg.TestCommand == "" {
					result.Unverified = true
				}

				resMu.Lock()
				results[idx] = result
				resMu.Unlock()

				events <- Event{Type: EventTaskFinished, InstanceID: t.InstanceID, Result: result}

				costMu.Lock()
				currentCost += result.CostUSD
				if e.cfg.MaxCost > 0 && currentCost >= e.cfg.MaxCost {
					events <- Event{
						Type:    EventBudgetExceeded,
						Message: fmt.Sprintf("Max cost of $%.2f exceeded (Current: $%.2f). Cancelling remaining tasks.", e.cfg.MaxCost, currentCost),
					}
					// Note: we can't easily cancel gCtx from here without passing a cancel func down,
					// but returning an error from errgroup cancels the context for us.
					costMu.Unlock()
					return fmt.Errorf("budget exceeded")
				}
				costMu.Unlock()

				return nil
			})
		}

		// Wait for all tasks to finish
		_ = g.Wait()
	}()

	return events, results, nil
}

package engine_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/engine"
	"github.com/precedent-cli/precedent/internal/types"
)

// mockAdapter simulates different agent behaviors.
type mockAdapter struct {
	mu          sync.Mutex
	delay       time.Duration
	err         error
	returnNil   bool
	costPerTask float64
}

func (m *mockAdapter) Name() string      { return "mock" }
func (m *mockAdapter) IsInstalled() bool { return true }
func (m *mockAdapter) Run(ctx context.Context, workDir string, taskPrompt string, task types.Task) (*adapters.AgentResult, error) {
	m.mu.Lock()
	delay := m.delay
	retErr := m.err
	retNil := m.returnNil
	cost := m.costPerTask
	m.mu.Unlock()

	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return &adapters.AgentResult{Error: ctx.Err()}, ctx.Err()
	}

	if retNil {
		return nil, retErr
	}
	if retErr != nil {
		return &adapters.AgentResult{Error: retErr}, retErr
	}

	return &adapters.AgentResult{
		CostUSD:  cost,
		Duration: delay,
	}, nil
}

func createTestRepo(t *testing.T) (string, string) {
	t.Helper()
	repoPath := t.TempDir()

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.name", "Test User")
	runGit("config", "user.email", "test@example.com")

	f, _ := os.Create(filepath.Join(repoPath, "README.md"))
	f.Close()

	runGit("add", "README.md")
	runGit("commit", "-m", "initial commit")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get commit: %v", err)
	}

	return repoPath, string(out[:len(out)-1]) // trim newline
}

func TestEngine_Success(t *testing.T) {
	repo, commit := createTestRepo(t)
	tasks := []types.Task{
		{InstanceID: "task1", BaseCommit: commit},
		{InstanceID: "task2", BaseCommit: commit},
	}

	cfg := engine.Config{
		RepoPath:    repo,
		WorktreeDir: filepath.Join(t.TempDir(), "wt"),
		Concurrency: 2,
		TaskTimeout: 5 * time.Second,
	}

	mock := &mockAdapter{}
	eng := engine.New(cfg, mock)

	events, results, err := eng.Run(context.Background(), tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var finished int
	for e := range events {
		if e.Type == engine.EventTaskFinished {
			finished++
			if e.Result == nil || e.Result.Error != nil {
				t.Errorf("task %s failed: %v", e.InstanceID, e.Result)
			}
		}
	}
	if finished != 2 {
		t.Errorf("expected 2 finished tasks, got %d", finished)
	}

	if len(results) != 2 || results[0].Error != nil || results[1].Error != nil {
		t.Errorf("results array is incorrect")
	}
}

func TestEngine_Timeout(t *testing.T) {
	repo, commit := createTestRepo(t)
	tasks := []types.Task{{InstanceID: "task1", BaseCommit: commit}}

	cfg := engine.Config{
		RepoPath:    repo,
		WorktreeDir: filepath.Join(t.TempDir(), "wt"),
		Concurrency: 1,
		TaskTimeout: 100 * time.Millisecond,
	}

	// Adapter takes longer than the timeout
	mock := &mockAdapter{delay: 500 * time.Millisecond}
	eng := engine.New(cfg, mock)

	events, results, _ := eng.Run(context.Background(), tasks)

	var resErr error
	for e := range events {
		if e.Type == engine.EventTaskFinished {
			resErr = e.Result.Error
		}
	}

	if resErr != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded, got: %v", resErr)
	}
	if results[0].Error != context.DeadlineExceeded {
		t.Errorf("results array should reflect timeout")
	}
}

func TestEngine_BudgetExceeded(t *testing.T) {
	repo, commit := createTestRepo(t)
	// 5 tasks, each costs 1.0. Max cost is 1.5.
	// At most 2 tasks should complete before the budget exceeds and the rest are cancelled.
	var tasks []types.Task
	for i := 0; i < 5; i++ {
		tasks = append(tasks, types.Task{InstanceID: fmt.Sprintf("task%d", i), BaseCommit: commit})
	}

	cfg := engine.Config{
		RepoPath:    repo,
		WorktreeDir: filepath.Join(t.TempDir(), "wt"),
		Concurrency: 1, // run sequentially to easily test budget accumulation
		TaskTimeout: 5 * time.Second,
		MaxCost:     1.5,
	}

	mock := &mockAdapter{costPerTask: 1.0}
	eng := engine.New(cfg, mock)

	events, _, _ := eng.Run(context.Background(), tasks)

	budgetEventSeen := false
	for e := range events {
		if e.Type == engine.EventBudgetExceeded {
			budgetEventSeen = true
		}
	}

	if !budgetEventSeen {
		t.Errorf("expected EventBudgetExceeded")
	}
}

func TestEngine_NilResultGuard(t *testing.T) {
	repo, commit := createTestRepo(t)
	tasks := []types.Task{{InstanceID: "task1", BaseCommit: commit}}

	cfg := engine.Config{
		RepoPath:    repo,
		WorktreeDir: filepath.Join(t.TempDir(), "wt"),
		Concurrency: 1,
		TaskTimeout: 5 * time.Second,
	}

	// Adapter returns a nil pointer to test engine's guard
	mock := &mockAdapter{returnNil: true, err: fmt.Errorf("some internal panic")}
	eng := engine.New(cfg, mock)

	events, results, _ := eng.Run(context.Background(), tasks)

	for e := range events {
		if e.Type == engine.EventTaskFinished {
			if e.Result == nil {
				t.Fatalf("Engine emitted nil result")
			}
			if e.Result.Error.Error() != "some internal panic" {
				t.Errorf("expected synthesized error, got: %v", e.Result.Error)
			}
		}
	}

	if results[0] == nil {
		t.Fatalf("results array has nil")
	}
}

package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// GitIsolation manages a detached git worktree for safe task execution.
type GitIsolation struct {
	RepoPath    string
	WorktreeDir string
	BaseCommit  string
}

// NewGitIsolation creates a new GitIsolation manager.
func NewGitIsolation(repoPath, worktreeDir, baseCommit string) *GitIsolation {
	return &GitIsolation{
		RepoPath:    repoPath,
		WorktreeDir: worktreeDir,
		BaseCommit:  baseCommit,
	}
}

// Setup creates the worktree.
func (g *GitIsolation) Setup(ctx context.Context) error {
	// git worktree add --detach <dir> <commit>
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", g.WorktreeDir, g.BaseCommit)
	cmd.Dir = g.RepoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to setup worktree: %v (output: %s)", err, out)
	}
	return nil
}

// Teardown forcefully removes the worktree.
func (g *GitIsolation) Teardown(ctx context.Context) error {
	// 1. Remove the directory forcefully via git
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", g.WorktreeDir)
	cmd.Dir = g.RepoPath
	_ = cmd.Run() // Ignore errors, fallback to os.RemoveAll

	// 2. Fallback to OS removal just in case
	return os.RemoveAll(g.WorktreeDir)
}

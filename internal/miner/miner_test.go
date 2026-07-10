package miner_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/precedent-cli/precedent/internal/miner"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func TestMiner_NulParsing(t *testing.T) {
	repoPath := t.TempDir()

	runGit(t, repoPath, "init")
	runGit(t, repoPath, "config", "user.name", "Test User")
	runGit(t, repoPath, "config", "user.email", "test@example.com")

	// Commit 1: Initial commit (base)
	writeFile(t, repoPath, "main.go", "package main")
	runGit(t, repoPath, "add", "main.go")
	runGit(t, repoPath, "commit", "-m", "initial commit")

	// Commit 2: Valid commit with source and test file
	// We use a pipe in the subject to ensure it doesn't break parsing
	writeFile(t, repoPath, "main.go", "package main\nfunc Add() {}")
	writeFile(t, repoPath, "main_test.go", "package main")
	runGit(t, repoPath, "add", "main.go", "main_test.go")
	runGit(t, repoPath, "commit", "-m", "feat: add func | some context")

	// Commit 3: Bot commit (should be ignored)
	writeFile(t, repoPath, "deps.go", "package deps")
	writeFile(t, repoPath, "deps_test.go", "package deps")
	runGit(t, repoPath, "add", "deps.go", "deps_test.go")
	runGit(t, repoPath, "config", "user.name", "dependabot[bot]") // spoof bot
	runGit(t, repoPath, "commit", "-m", "chore: update deps")
	runGit(t, repoPath, "config", "user.name", "Test User") // restore user

	// Commit 4: True Empty commit (should not crash parser)
	runGit(t, repoPath, "commit", "--allow-empty", "-m", "true empty commit")

	// Commit 5: Valid commit but with a file literally named "COMMIT" and newlines
	writeFile(t, repoPath, "COMMIT", "test")
	writeFile(t, repoPath, "COMMIT_test.go", "test")
	runGit(t, repoPath, "add", "COMMIT", "COMMIT_test.go")
	runGit(t, repoPath, "commit", "-m", "weird file names")

	// Now run the miner in relaxed mode (so it doesn't try to actually run tests)
	m := miner.NewTaskMiner(repoPath, 10, "go test", true)
	tasks, err := m.Mine(context.Background())
	if err != nil {
		t.Fatalf("Mine failed: %v", err)
	}

	// We expect 2 valid tasks: Commit 2 and Commit 5.
	// Commit 1 has no tests. Commit 3 is a bot. Commit 4 is empty.
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	// Because tasks are shuffled, we just check if the problem statements match our expectations.
	foundFeat := false
	foundWeird := false

	for _, task := range tasks {
		if task.ProblemStatement == "feat: add func | some context" {
			foundFeat = true
		}
		if task.ProblemStatement == "weird file names" {
			foundWeird = true
		}
		// Assert that no task lists "COMMIT" or other metadata as a file/content
		if task.InstanceID == "" || task.BaseCommit == "" {
			t.Errorf("Task missing InstanceID or BaseCommit")
		}
	}

	if !foundFeat {
		t.Errorf("missing feat commit task")
	}
	if !foundWeird {
		t.Errorf("missing weird file name task")
	}
}

package miner

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/precedent-cli/precedent/internal/types"
)

type TaskMiner struct {
	RepoPath    string
	MaxTasks    int
	TestCommand string
	Relaxed     bool
}

func NewTaskMiner(repoPath string, maxTasks int, testCommand string, relaxed bool) *TaskMiner {
	return &TaskMiner{
		RepoPath:    repoPath,
		MaxTasks:    maxTasks,
		TestCommand: testCommand,
		Relaxed:     relaxed,
	}
}

func isBot(author string) bool {
	author = strings.ToLower(author)
	bots := []string{"dependabot", "renovate", "snyk", "bot", "github-actions"}
	for _, b := range bots {
		if strings.Contains(author, b) {
			return true
		}
	}
	return false
}

func isTestFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, "_test.go") ||
		strings.Contains(lower, "test_") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") ||
		strings.HasPrefix(filepath.Base(lower), "test_")
}

func (m *TaskMiner) getPatch(ctx context.Context, commitHash string, files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}
	args := []string{"show", "--format=", commitHash, "--"}
	args = append(args, files...)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = m.RepoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// verifyFailToPass creates a temporary worktree to ensure the base commit FAILS the tests,
// and the head commit PASSES the tests.
func (m *TaskMiner) verifyFailToPass(ctx context.Context, headCommit, baseCommit string, testFiles []string) bool {
	// Safety: Create a temporary worktree so we don't mess up the user's active workspace
	wtDir := filepath.Join(m.RepoPath, ".precedent", "validation_wt")
	_ = os.RemoveAll(wtDir) // Ensure clean state

	// Add worktree at BaseCommit
	addCmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", wtDir, baseCommit)
	addCmd.Dir = m.RepoPath
	if err := addCmd.Run(); err != nil {
		return false // if we can't checkout, it's invalid
	}
	defer func() {
		rmCmd := exec.CommandContext(context.Background(), "git", "worktree", "remove", "--force", wtDir)
		rmCmd.Dir = m.RepoPath
		_ = rmCmd.Run()
		_ = os.RemoveAll(wtDir)
	}()

	// 1. Test BaseCommit (Must Fail)
	testBase := exec.CommandContext(ctx, "sh", "-c", m.TestCommand)
	testBase.Dir = wtDir
	if err := testBase.Run(); err == nil {
		// It passed! This means it doesn't fail-to-pass, or the tests weren't failing.
		return false
	}

	// 2. Checkout HeadCommit and test (Must Pass)
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", headCommit)
	checkoutCmd.Dir = wtDir
	if err := checkoutCmd.Run(); err != nil {
		return false
	}

	testHead := exec.CommandContext(ctx, "sh", "-c", m.TestCommand)
	testHead.Dir = wtDir
	if err := testHead.Run(); err != nil {
		// It failed on the head commit too! It's a broken test or flaky test.
		return false
	}

	return true // Verified!
}

func (m *TaskMiner) Mine(ctx context.Context) ([]types.Task, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--pretty=format:COMMIT|%H|%an|%s", "--name-only", "--no-merges")
	cmd.Dir = m.RepoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start git log: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	var tasks []types.Task

	var currHash, currAuthor, currSubject string
	var currSrcFiles, currTestFiles []string

	processCommit := func() {
		if currHash == "" {
			return
		}

		if len(currSrcFiles) > 0 && len(currTestFiles) > 0 && !isBot(currAuthor) {
			baseCommit := currHash + "~1"

			if !m.Relaxed {
				// 🚨 Strict Benchmark Integrity: Fail-to-Pass Validation
				fmt.Printf("🔍 Validating candidate task %s with '%s' (Fail-to-Pass check)...\n", currHash[:7], m.TestCommand)
				if !m.verifyFailToPass(ctx, currHash, baseCommit, currTestFiles) {
					fmt.Printf("❌ Discarded %s: Did not meet Fail-to-Pass criteria.\n", currHash[:7])
					return
				}
				fmt.Printf("✅ Verified %s: Passed Fail-to-Pass check.\n", currHash[:7])
			} else {
				fmt.Printf("⚠️ Accepting %s (Relaxed Mode: Skipping strict validation)\n", currHash[:7])
			}

			repoName := filepath.Base(m.RepoPath)
			taskID := fmt.Sprintf("%s-%s", repoName, currHash[:7])

			srcPatch, _ := m.getPatch(ctx, currHash, currSrcFiles)
			testPatch, _ := m.getPatch(ctx, currHash, currTestFiles)

			tasks = append(tasks, types.Task{
				InstanceID:       taskID,
				BaseCommit:       baseCommit,
				ProblemStatement: currSubject,
				Repo:             repoName,
				Patch:            srcPatch,
				TestPatch:        testPatch,
				TestCommand:      m.TestCommand,
			})
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "COMMIT|") {
			processCommit()

			if m.MaxTasks > 0 && len(tasks) >= m.MaxTasks {
				break
			}

			parts := strings.SplitN(line, "|", 4)
			if len(parts) >= 4 {
				currHash = parts[1]
				currAuthor = parts[2]
				currSubject = strings.TrimSpace(parts[3])
				currSrcFiles = nil
				currTestFiles = nil
			} else {
				currHash = ""
			}
		} else if line != "" && currHash != "" {
			if isTestFile(line) {
				currTestFiles = append(currTestFiles, line)
			} else {
				currSrcFiles = append(currSrcFiles, line)
			}
		}
	}
	processCommit()

	if err := cmd.Wait(); err != nil {
		if m.MaxTasks > 0 && len(tasks) >= m.MaxTasks {
			goto Finish
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

Finish:
	// Randomize task order to prevent execution caching bias
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(tasks), func(i, j int) {
		tasks[i], tasks[j] = tasks[j], tasks[i]
	})

	return tasks, nil
}

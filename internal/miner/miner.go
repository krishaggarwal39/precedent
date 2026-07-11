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

	"github.com/precedent-cli/precedent/internal/runner"
	"github.com/precedent-cli/precedent/internal/types"
)

// TaskMiner is responsible for extracting benchmark tasks from a git repository.
type TaskMiner struct {
	RepoPath    string
	MaxTasks    int
	TestCommand string
	Relaxed     bool
	TaskTimeout time.Duration
}

// NewTaskMiner creates a new TaskMiner instance with the given configuration.
func NewTaskMiner(repoPath string, maxTasks int, testCommand string, relaxed bool, taskTimeout time.Duration) *TaskMiner {
	return &TaskMiner{
		RepoPath:    repoPath,
		MaxTasks:    maxTasks,
		TestCommand: testCommand,
		Relaxed:     relaxed,
		TaskTimeout: taskTimeout,
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
	wtDir, err := os.MkdirTemp("", "precedent-val-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(wtDir)

	isolation := runner.NewGitIsolation(m.RepoPath, wtDir, baseCommit)
	if err := isolation.Setup(ctx); err != nil {
		return false
	}
	defer func() {
		_ = isolation.Teardown(context.Background())
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, m.TaskTimeout)
	defer cancel()

	// 1. Test BaseCommit (Must Fail)
	testBase := exec.CommandContext(timeoutCtx, "sh", "-c", m.TestCommand)
	testBase.Dir = wtDir
	if err := testBase.Run(); err == nil {
		// It passed! This means it doesn't fail-to-pass, or the tests weren't failing.
		return false
	}

	// 2. Checkout HeadCommit and test (Must Pass)
	checkoutCmd := exec.CommandContext(timeoutCtx, "git", "checkout", headCommit)
	checkoutCmd.Dir = wtDir
	if err := checkoutCmd.Run(); err != nil {
		return false
	}

	testHead := exec.CommandContext(timeoutCtx, "sh", "-c", m.TestCommand)
	testHead.Dir = wtDir
	if err := testHead.Run(); err != nil {
		// It failed on the head commit too! It's a broken test or flaky test.
		return false
	}

	return true // Verified!
}

// Mine iterates through the git history to extract and validate fail-to-pass tasks.
func (m *TaskMiner) Mine(ctx context.Context) ([]types.Task, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--pretty=format:COMMIT%x00%H%x00%an%x00%s%x00", "--name-only", "--no-merges", "-z")
	cmd.Dir = m.RepoPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start git log: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	const maxCapacity = 50 * 1024 * 1024 // 50MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.IndexByte(string(data), 0); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

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
			})
		}
	}

	state := 0 // 0=expect COMMIT, 1=hash, 2=author, 3=subject, 4=files
	for scanner.Scan() {
		token := scanner.Text()

		if state == 0 {
			if token == "COMMIT" {
				processCommit()
				if m.MaxTasks > 0 && len(tasks) >= m.MaxTasks {
					break
				}
				currHash, currAuthor, currSubject = "", "", ""
				currSrcFiles, currTestFiles = nil, nil
				state = 1
			}
			continue
		}

		switch state {
		case 1:
			currHash = token
			state = 2
		case 2:
			currAuthor = token
			state = 3
		case 3:
			currSubject = token
			state = 4
		case 4:
			if token == "" {
				state = 0 // End of file list
				continue
			}
			if token == "COMMIT" {
				// Edge case: git omitted the empty separator token
				processCommit()
				if m.MaxTasks > 0 && len(tasks) >= m.MaxTasks {
					break
				}
				currHash, currAuthor, currSubject = "", "", ""
				currSrcFiles, currTestFiles = nil, nil
				state = 1
				continue
			}
			filename := strings.TrimPrefix(token, "\n")
			if filename == "" {
				continue
			}
			if isTestFile(filename) {
				currTestFiles = append(currTestFiles, filename)
			} else {
				currSrcFiles = append(currSrcFiles, filename)
			}
		}
	}
	processCommit()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse git log history: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if m.MaxTasks <= 0 || len(tasks) < m.MaxTasks {
			return nil, fmt.Errorf("git log failed: %w", err)
		}
	}

	// Randomize task order to prevent execution caching bias
	rand.Shuffle(len(tasks), func(i, j int) {
		tasks[i], tasks[j] = tasks[j], tasks[i]
	})

	return tasks, nil
}

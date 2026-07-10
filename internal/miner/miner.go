package miner

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/precedent-cli/precedent/internal/types"
)

// TaskMiner scans a git repository to find commits that represent valid tasks.
type TaskMiner struct {
	RepoPath string
	MaxTasks int
}

// NewTaskMiner creates a new TaskMiner.
func NewTaskMiner(repoPath string, maxTasks int) *TaskMiner {
	return &TaskMiner{
		RepoPath: repoPath,
		MaxTasks: maxTasks,
	}
}

// isBot returns true if the author looks like a bot.
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

// isTestFile attempts to heuristically determine if a file is a test file.
func isTestFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, "_test.go") ||
		strings.Contains(lower, "test_") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") ||
		strings.HasPrefix(filepath.Base(lower), "test_")
}

// Mine runs git log and extracts tasks based on SWE-bench fail-to-pass methodology.
func (m *TaskMiner) Mine(ctx context.Context) ([]types.Task, error) {
	// Execute git log streaming the output.
	// Format: COMMIT|hash|author_name|subject
	// Followed by the list of modified files due to --name-only
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
	var currModifiesSrc, currModifiesTest bool

	processCommit := func() {
		if currHash == "" {
			return
		}
		// A valid fix modifies both source code and tests, and is not a bot.
		if currModifiesSrc && currModifiesTest && !isBot(currAuthor) {
			repoName := filepath.Base(m.RepoPath)
			taskID := fmt.Sprintf("%s-%s", repoName, currHash[:7])

			tasks = append(tasks, types.Task{
				InstanceID:       taskID,
				BaseCommit:       currHash + "~1", // The state of the repo right before the fix
				ProblemStatement: currSubject,
				Repo:             repoName,
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
				currModifiesSrc = false
				currModifiesTest = false
			} else {
				currHash = ""
			}
		} else if line != "" && currHash != "" {
			// This is a modified file path
			if isTestFile(line) {
				currModifiesTest = true
			} else {
				currModifiesSrc = true
			}
		}
	}
	processCommit()

	// Wait for the command to finish, ignoring errors if we hit our max tasks limit.
	if err := cmd.Wait(); err != nil {
		if m.MaxTasks > 0 && len(tasks) >= m.MaxTasks {
			return tasks, nil
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return tasks, nil
}

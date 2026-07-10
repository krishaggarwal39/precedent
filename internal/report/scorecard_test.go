package report_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/report"
	"github.com/precedent-cli/precedent/internal/types"
)

func TestGenerateScorecard_Golden(t *testing.T) {
	tasks := []types.Task{
		{InstanceID: "task-pass", BaseCommit: "abc"},
		{InstanceID: "task-fail", BaseCommit: "def"},
		{InstanceID: "task-unverified", BaseCommit: "ghi"},
		{InstanceID: "task-skipped", BaseCommit: "jkl"},
	}

	results := []*adapters.AgentResult{
		{CostUSD: 0.10, Duration: 5 * time.Second, TotalTokens: 1000}, // PASS
		{Error: errors.New("compile error")},                          // FAIL
		{Unverified: true},                                            // UNVERIFIED
		nil,                                                           // SKIPPED
	}

	outPath := filepath.Join(t.TempDir(), "scorecard.html")

	err := report.GenerateScorecard(tasks, results, outPath)
	if err != nil {
		t.Fatalf("GenerateScorecard failed: %v", err)
	}

	outBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}
	outStr := string(outBytes)

	// We just check if the generated HTML contains the specific task entries
	// and handles the nil result gracefully (no panics).
	// Doing exact golden file matching on generated HTML is often brittle due to timestamps.

	expectedSubstrings := []string{
		"task-pass",
		"task-fail",
		"task-unverified",
		"task-skipped",
		"$0.10",
		"250",
		"pass",
		"fail",
		"unverified",
		"skipped",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(outStr, sub) {
			t.Errorf("Scorecard is missing expected substring: %q", sub)
		}
	}

	if strings.Contains(outStr, "http://") || strings.Contains(outStr, "https://") {
		t.Errorf("Scorecard contains external resources (http/https). It must be offline-safe.")
	}
}

package miner

import (
	"os"
	"path/filepath"
)

// DetectTestCommand inspects the repository root to determine the likely test framework.
func DetectTestCommand(repoPath string) string {
	// Node.js
	if _, err := os.Stat(filepath.Join(repoPath, "package.json")); err == nil {
		return "npm test"
	}

	// Go
	if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); err == nil {
		return "go test ./..."
	}

	// Python (Pytest is standard for data/AI repos)
	if _, err := os.Stat(filepath.Join(repoPath, "pytest.ini")); err == nil {
		return "pytest"
	}
	if _, err := os.Stat(filepath.Join(repoPath, "requirements.txt")); err == nil {
		return "pytest"
	}
	if _, err := os.Stat(filepath.Join(repoPath, "setup.py")); err == nil {
		return "pytest"
	}

	// Rust
	if _, err := os.Stat(filepath.Join(repoPath, "Cargo.toml")); err == nil {
		return "cargo test"
	}

	// Ruby
	if _, err := os.Stat(filepath.Join(repoPath, "Gemfile")); err == nil {
		return "rspec"
	}

	return "" // Could not auto-detect
}

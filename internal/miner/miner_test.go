package miner

import (
	"testing"
)

func TestIsBot(t *testing.T) {
	tests := []struct {
		author string
		want   bool
	}{
		{"dependabot[bot]", true},
		{"renovate", true},
		{"snyk", true},
		{"github-actions", true},
		{"John Doe", false},
		{"JaneBot", true}, // Contains "bot"
	}

	for _, tt := range tests {
		t.Run(tt.author, func(t *testing.T) {
			if got := isBot(tt.author); got != tt.want {
				t.Errorf("isBot(%q) = %v, want %v", tt.author, got, tt.want)
			}
		})
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"main_test.go", true},
		{"test_utils.py", true},
		{"app.test.js", true},
		{"app.spec.ts", true},
		{"utils/test_helper.rb", true},
		{"main.go", false},
		{"utils.py", false},
		{"app.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := isTestFile(tt.filename); got != tt.want {
				t.Errorf("isTestFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

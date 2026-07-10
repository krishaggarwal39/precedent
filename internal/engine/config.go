package engine

import "time"

// Config holds the configuration for a benchmark run.
type Config struct {
	RepoPath    string
	TasksDir    string
	WorktreeDir string
	MaxCost     float64
	Concurrency int
	DockerImage string
	TaskTimeout time.Duration
	TestCommand string
}

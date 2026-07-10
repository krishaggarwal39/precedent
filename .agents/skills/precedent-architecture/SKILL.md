---
name: precedent-architecture
description: Architecture guide for the Precedent CLI tool. Use when building components, writing adapters, or modifying the project structure.
---

## Project Structure
- Entry point: `main.go` → `cmd.Execute()`
- Commands: `cmd/root.go`, `cmd/init.go`, `cmd/run.go`
- Business logic: `internal/miner/`, `internal/runner/`, `internal/adapters/`, `internal/grader/`, `internal/report/`
- Templates: `templates/` (embedded via go:embed)

## Agent Adapter Interface
All adapters MUST implement:
- `Name() string`
- `Run(ctx context.Context, workDir string, taskPrompt string) (*AgentResult, error)`
- `IsInstalled() bool`

## Key Patterns
- Use `os/exec` for git operations, NOT go-git
- Use `--detach` flag for worktrees
- Parse agent output JSON for cost/tokens
- Always set process group on spawned agent processes
- Always defer worktree cleanup

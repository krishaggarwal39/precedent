# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - Initial Release

### Added
- **Core CLI:** `precedent init` and `precedent run` commands.
- **Task Miner:** Implements Databricks "Fail-to-Pass" validation metric. Automatically extracts valid AI tasks from git history.
- **Git Worktree Sandbox:** Isolated environments for AI execution to prevent repository corruption.
- **Agent Adapters:** Native support for Anthropic's `claude-code`.
- **BYOA (Bring Your Own Agent):** Added `.precedent/agents.yaml` configuration to support running any arbitrary CLI AI agent.
- **Premium UI:** Integrated Bubbletea interactive spinners for the CLI.
- **HTML Scorecard:** Zero-dependency HTML report generator summarizing benchmark pass rates and API costs. Offline-safe with embedded styles.
- **Cost Tracking:** Real cost parsing from Claude's JSON output with `--max-cost` safety nets.
- **Architecture Refactor:** Decoupled execution loop into the `internal/engine` package for robustness.
- **Security:** Strict 10-minute timeouts, process-group `SIGKILL`, NUL-byte git parsing, shell-injection defenses via `PRECEDENT_PROMPT`, and interactive Trust Boundary prompts.
- **Cross-Platform:** Full support for Linux, macOS, and Windows.

### Known Debt
- The compiled `precedent` binary still exists in earlier git history commits. Purging it via `git filter-repo` was deferred to avoid breaking existing clones.

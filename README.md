<h1 align="center">Precedent 🚀</h1>

<p align="center">
  <strong>The Ultimate Manual: Test AI Coding Agents on YOUR Project</strong><br>
  <a href="https://github.com/precedent-cli/precedent/actions"><img src="https://github.com/precedent-cli/precedent/workflows/ci/badge.svg" alt="CI Status"></a>
  <a href="https://goreportcard.com/report/github.com/precedent-cli/precedent"><img src="https://goreportcard.com/badge/github.com/precedent-cli/precedent" alt="Go Report Card"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

---

Welcome to the **Complete Precedent Manual**! Whether you are a beginner looking to test your first AI, or a senior engineer setting up a rigorous benchmark, this guide will explain everything in simple terms.

## 📖 Table of Contents
1. [What is Precedent?](#1-what-is-precedent)
2. [Quickstart](#2-quickstart)
3. [Bring Your Own Agent (BYOA) Guide](#3-bring-your-own-agent-byoa-guide)
4. [Scorecard Statuses](#4-scorecard-statuses)
5. [Security Model](#5-security-model)

---

## 1. What is Precedent?

Precedent is a simple terminal tool that benchmarks AI Coding Agents on your actual codebase. It finds old bugs you already fixed in your project's git history, drops the Agent into that old codebase, and checks if the Agent can autonomously fix the bug. 

**Methodology: Fail-to-Pass**
Precedent is strict. It only tests the AI on historical bugs where your tests *failed* before the bug was fixed, and *passed* after. No fake or flaky tests are allowed! At the end, it generates a self-contained, offline-safe `scorecard.html` showing you exactly which Agentic IDE is the smartest and cheapest.

---

## 2. Quickstart

### Installation

**macOS (Homebrew)**
```bash
brew tap precedent-cli/tap
brew install precedent
```

**Linux / Windows (Go)**
```bash
go install github.com/precedent-cli/precedent@latest
```

### Step A: Initialize the Benchmark
Go to any project folder on your computer that has a Git history and run:
```bash
cd your-project-folder
precedent init
```
* `--test-cmd "COMMAND"` : Tell Precedent how you run your tests (e.g., `"npm test"` or `"go test ./..."`). Precedent auto-detects this if omitted.
* `--relaxed` : Skip the strict Fail-to-Pass test verification. Use this if your older commits have broken tests.

### Step B: Run the AI Race
Now, let's ask Claude to fix those bugs.
```bash
precedent run --agent claude --docker-image node:18 --yes
```

**Run Flags:**
* `--agent NAME` : Choose which AI to use (default: `claude`).
* `--docker-image IMAGE` : The Docker image to run tests in (e.g., `node:18`). If empty, tests run on the host.
* `--concurrency N` : Run `N` tasks at the same time (0 = auto-detect).
* `--max-cost AMOUNT` : Maximum total cost in USD before aborting (Default: 5.0). Currently supports the `claude` adapter.
* `--test-cmd COMMAND` : The test command to verify fixes.
* `--task-timeout DURATION` : Timeout per task (Default: 10m).
* `--yes` : Skip the interactive security confirmation prompt.

---

## 3. Bring Your Own Agent (BYOA) Guide

You can test ANY AI tool by creating `.precedent/agents.yaml` in your project folder.

### The YAML Format
```yaml
aider:
  # The prompt is delivered securely via the PRECEDENT_PROMPT environment variable.
  command: 'aider --message "$PRECEDENT_PROMPT" --yes'
```

**IMPORTANT:** For security reasons against prompt injection, you must NOT interpolate the prompt directly. The task's bug description is injected via the `PRECEDENT_PROMPT` environment variable, and the worktree path via `PRECEDENT_WORKTREE`. Use `$PRECEDENT_PROMPT` safely in your command strings.

---

## 4. Scorecard Statuses

The generated HTML scorecard uses the following statuses:
* **PASS**: The agent successfully fixed the bug, and the test command passed.
* **FAIL**: The agent failed to fix the bug (the test command failed, or the agent errored out/timed out).
* **UNVERIFIED**: The agent finished its work, but no test command was provided to verify it.
* **SKIPPED**: The task was skipped or aborted (e.g., if you quit early).

---

## 5. Security Model

Precedent executes untrusted, AI-generated code. Our security model handles this via:

* **Trust Boundary**: Test commands run against AI-modified code. Before execution, Precedent shows a security confirmation prompt detailing the agent, test command, and whether it runs in Docker or on the host.
* **Docker Sandboxing**: When using `--docker-image`, tests run in a locked-down container (`--network=none --security-opt=no-new-privileges --cap-drop=ALL`).
* **Host Execution Warning**: If `--docker-image` is omitted, tests run directly on your host machine. Precedent explicitly warns you about this.
* **Known Limitations**: The mounted worktree is fully writable, and we do not apply a custom seccomp profile.

Always use `--docker-image` for untrusted execution.

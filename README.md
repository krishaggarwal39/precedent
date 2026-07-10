<h1 align="center">Precedent 🚀</h1>

<p align="center">
  <strong>The Ultimate Manual: Test AI Coding Agents on YOUR Project</strong><br>
</p>

---

Welcome to the **Complete Precedent Manual**! Whether you are a beginner looking to test your first AI, or a senior engineer setting up a Databricks-level benchmark, this guide will explain everything in simple terms.

## 📖 Table of Contents
1. [What is Precedent?](#1-what-is-precedent)
2. [Key Features](#2-key-features)
3. [Installation](#3-installation)
4. [Step-by-Step Tutorial](#4-step-by-step-tutorial)
5. [Complete CLI Reference (All Commands)](#5-complete-cli-reference)
6. [Bring Your Own Agent (BYOA) Guide](#6-bring-your-own-agent-byoa-guide)
7. [Frequently Asked Questions (FAQ)](#7-frequently-asked-questions)

---

## 1. What is Precedent?

There is a huge difference between a **Raw AI Model** (like Gemini or GPT-4 answering a text prompt) and an **Agentic IDE / Autonomous Agent** (like Antigravity, Aider, or Claude Code that can read your files, run terminal commands, and write code on their own). 

Standard benchmarks test raw models, but **Precedent tests Agents.**

Imagine you want to hire an AI Agent to write code for your project. You might be confused: 
* *"Is Antigravity better than Aider at navigating MY specific codebase?"* 
* *"Can this AI actually find the bug across multiple files and fix it without breaking other things?"* 
* *"How much API cost will it burn?"* 

**Precedent** is a simple terminal tool that answers these questions. It finds old bugs you already fixed in your project's git history, drops the Agent into that old codebase, and checks if the Agent can autonomously fix the bug! At the end, it generates a beautiful `scorecard.html` showing you exactly which Agentic IDE is the smartest and cheapest.

---

## 2. Key Features

Precedent is packed with Enterprise-grade features, simplified for everyday use:

* 🛡️ **Zero-Space Docker Sandboxing:** We run the AI's code inside a secure Docker container (like a virtual box). If the AI hallucinates and writes a dangerous command, it cannot harm your laptop.
* 🚦 **Fail-to-Pass Verification:** Precedent is strict. It only tests the AI on historical bugs where your tests *failed* before the bug was fixed, and *passed* after. No fake or flaky tests are allowed!
* 🧠 **Smart Auto-Concurrency:** If you have an 8-core Macbook, Precedent automatically runs multiple AI tests at the same time to finish faster without crashing your computer.
* 💸 **Max-Cost Safety Net:** Never accidentally burn your API wallet! Precedent automatically stops the benchmark if the AI spends more than your allowed budget (default $5).
* 🔌 **Bring Your Own Agent (BYOA):** Test literally *any* AI tool in the world using a simple YAML file. 

---

## 3. Installation

Precedent is a single, lightning-fast binary. You can install it on any operating system:

### 🍎 macOS
Using Homebrew (Recommended):
```bash
brew tap precedent-cli/tap
brew install precedent
```
*Alternatively, you can download the binary from the [Releases page](https://github.com/precedent-cli/precedent/releases).*

### 🐧 Linux
Using cURL to install the latest pre-compiled binary:
```bash
curl -sL https://raw.githubusercontent.com/precedent-cli/precedent/main/install.sh | bash
```
*Or, download the `.deb` / `.rpm` package directly from our [Releases page](https://github.com/precedent-cli/precedent/releases).*

### 🪟 Windows
Using Scoop (Run in PowerShell):
```powershell
scoop bucket add precedent https://github.com/precedent-cli/scoop-bucket.git
scoop install precedent
```
*Alternatively, download the `.exe` file from the [Releases page](https://github.com/precedent-cli/precedent/releases).*

### 🛠️ For Developers (All Platforms)
If you already have Go installed, you can easily build it from source:
```bash
go install github.com/precedent-cli/precedent@latest
```
*(Make sure `$(go env GOPATH)/bin` is in your system's PATH).*

---

## 4. Step-by-Step Tutorial

### Step A: Initialize the Benchmark
Go to any project folder on your computer that has a Git history and run:
```bash
cd your-project-folder
precedent init
```
*(Precedent is smart! It automatically detects if your project uses `npm test`, `pytest`, or `go test`. You don't even need to tell it.)*

*Wait for a few seconds. Precedent is scanning your git history to find exactly 10 high-quality bugs.*

### Step B: Run the AI Race
Now, let's ask Claude to fix those bugs. (Make sure you have Docker running in the background for security).
```bash
precedent run --agent claude --docker-image node:18
```
*Precedent will show a beautiful loading screen while the AI works.*

### Step C: View the Results
When it finishes, open the newly created file:
```bash
open scorecard.html
```
*You will see a dashboard with the AI's success rate and total cost!*

---

## 5. Complete CLI Reference

Here is a list of every command and flag you can use to customize Precedent:

### `precedent init`
Scans your git history and prepares the benchmark tasks.
* `--test-cmd "COMMAND"` : **(Highly Recommended)** Tell Precedent how you run your tests (e.g., `"npm test"` or `"pytest"`). Precedent uses this to verify the bugs.
* `--relaxed` : Skip the strict Fail-to-Pass test verification. Use this if your older commits have broken tests.

### `precedent run`
Executes the benchmark by sending tasks to the AI.
* `--agent NAME` : Choose which AI to use (default: `claude`).
* `--docker-image IMAGE` : **(Crucial for Security)** The Docker image to run tests in (e.g., `node:18` or `python:3.10`). If left empty, tests run directly on your laptop (Not recommended).
* `--concurrency N` : Run `N` tasks at the same time. (Set to `0` to let Precedent auto-detect your CPU cores).
* `--max-cost AMOUNT` : Maximum total cost in USD before aborting to protect your wallet. (Default is `5.0`. Set to `0` for unlimited).

---

## 6. Bring Your Own Agent (BYOA) Guide

You can test ANY AI tool! Just create a file named `.precedent/agents.yaml` inside your project folder.

### The YAML Format
Here is how you add custom agents:

```yaml
aider:
  # {{PROMPT}} will automatically be replaced with the bug description
  command: "aider --message '{{PROMPT}}' --yes"
  
  # Optional: Tell Precedent how to read the cost from the AI's terminal output using Regex
  parse_cost: "Total cost: \\$([0-9.]+)"

ollama:
  # Test a free, local AI!
  command: "ollama run llama3 '{{PROMPT}}'"
```

**How to use it:**
Once the file is saved, just run:
```bash
precedent run --agent aider --docker-image node:18
```

---

## 7. Prior Art & Credits

Precedent was heavily inspired by the incredible work done by the academic and open-source communities to rigorously evaluate LLMs in software engineering:
* **[SWE-bench](https://www.swebench.com/):** For pioneering the framework of evaluating agents on real-world GitHub issues.
* **Databricks:** For their research on evaluating autonomous AI tools in isolated sandboxes.

Precedent aims to bring this Databricks/SWE-bench level of rigor to the everyday developer's local environment.

---

## 8. Frequently Asked Questions (FAQ)

**Q: Will the AI mess up my current code?**
No! Precedent creates a hidden folder (`.precedent/worktrees`) to test the AI. Your actual project files are 100% safe and untouched.

**Q: What if the AI gets stuck in an infinite loop?**
We have a strict **10-minute timeout** per task. If the AI doesn't finish, Precedent forcefully kills it and marks it as a FAIL.

**Q: Why did a task say "Discarded" during initialization?**
Precedent is very strict. If a bug fix in your history doesn't have a clear failing test that passes after the fix, Precedent throws it away. We only test the AI on 100% verified bugs.

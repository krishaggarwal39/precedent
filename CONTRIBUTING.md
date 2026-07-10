# Contributing to Precedent

First off, thank you for considering contributing to Precedent! We want to make testing AI agents as seamless and rigorous as possible.

## How to Add a New Agent Adapter

The easiest way to contribute is by adding support for a new AI Agent (like a new CLI tool). You do NOT need to write Go code to do this!

1. Install the agent locally and figure out the exact CLI command to run it headlessly.
2. Create a test configuration in `.precedent/agents.yaml`:
   ```yaml
   new-agent-name:
     command: "new-agent run --prompt '{{PROMPT}}'"
     parse_cost: "Cost: \\$([0-9.]+)"
   ```
3. Run a benchmark locally to ensure the agent correctly receives the prompt and edits the files.
4. Open an Issue using our `Add Agent Adapter` template and paste your working YAML snippet.
5. Our team will verify it and add it to the official Precedent documentation!

## Developing the Go Codebase

If you want to contribute to the core Go engine (e.g., improving the Git Miner, enhancing Docker Sandboxing, or fixing bugs):

1. **Fork the repo** and clone it locally.
2. **Run tests:** Ensure everything passes by running `go vet ./...` and `go test -race ./...`
3. **Format code:** Run `gofmt -w .` before submitting a PR.
4. **Submit a Pull Request** with a clear explanation of what you fixed or improved.

Please ensure that any new features include appropriate tests. We strictly adhere to the fail-to-pass methodology for benchmark integrity.

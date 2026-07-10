# Project Rules

## Go Development
- Use Go 1.22+ features (range over int, etc.)
- Follow standard Go project layout: cmd/, internal/, pkg/
- Use `cobra` for CLI commands and `viper` for configuration
- Always handle errors explicitly — no blank `_` for errors
- Use `context.Context` for cancellation propagation

## Code Quality
- Run `go vet ./...` and `go fmt ./...` before committing
- Keep functions under 50 lines where possible
- Use table-driven tests
- Document all exported types and functions

## Git
- Use conventional commits (feat:, fix:, chore:, docs:)
- Keep commits atomic and focused

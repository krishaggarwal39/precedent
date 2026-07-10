package adapters

import (
	"fmt"

	"github.com/precedent-cli/precedent/internal/config"
)

var registry = make(map[string]func() AgentAdapter)

// Register adds an agent adapter factory to the registry.
func Register(name string, factory func() AgentAdapter) {
	registry[name] = factory
}

// Resolve finds an adapter by name. It first checks the static registry (e.g., claude, dummy),
// then falls back to the dynamic YAML configuration.
func Resolve(name string, customConfigs map[string]config.AgentConfig) (AgentAdapter, error) {
	// 1. Check static registry (claude, dummy, etc.)
	if factory, exists := registry[name]; exists {
		return factory(), nil
	}
	// Also map "claude-code" to "claude" for backward compatibility if "claude" is registered
	if name == "claude-code" {
		if factory, exists := registry["claude"]; exists {
			return factory(), nil
		}
	}

	// 2. Check dynamic YAML configurations
	if cfg, exists := customConfigs[name]; exists {
		return &YamlAdapter{
			AgentName: name,
			Config:    cfg,
		}, nil
	}

	return nil, fmt.Errorf("unknown agent: %s. Define it in .precedent/agents.yaml or use a built-in agent", name)
}

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	Command string   `yaml:"command"`
	Env     []string `yaml:"env,omitempty"`
}

type AgentsFile struct {
	Agents map[string]AgentConfig `yaml:"agents"`
}

// LoadAgentsConfig parses the .precedent/agents.yaml file if it exists.
func LoadAgentsConfig(path string) (map[string]AgentConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]AgentConfig), nil // return empty if file doesn't exist
		}
		return nil, fmt.Errorf("reading agents config: %w", err)
	}

	var file AgentsFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, fmt.Errorf("parsing agents.yaml: %w", err)
	}

	return file.Agents, nil
}

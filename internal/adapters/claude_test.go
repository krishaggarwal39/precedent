package adapters

import (
	"testing"
)

func TestParseClaudeOutput(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		wantCost   float64
		wantTokens int
		wantErr    bool
	}{
		{
			name: "valid json",
			input: []byte(`{
				"total_cost_usd": 0.15,
				"usage": {
					"input_tokens": 1000,
					"output_tokens": 500
				},
				"is_error": false,
				"result": "success"
			}`),
			wantCost:   0.15,
			wantTokens: 1500,
			wantErr:    false,
		},
		{
			name: "valid json with unknown fields",
			input: []byte(`{
				"total_cost_usd": 1.25,
				"usage": {
					"input_tokens": 200,
					"output_tokens": 100,
					"cache_hits": 50
				},
				"is_error": false,
				"result": "success",
				"unknown_field": "hello"
			}`),
			wantCost:   1.25,
			wantTokens: 300,
			wantErr:    false,
		},
		{
			name:       "invalid json",
			input:      []byte(`{ "total_cost_usd": 0.15, broken`),
			wantCost:   0,
			wantTokens: 0,
			wantErr:    true,
		},
		{
			name:       "empty input",
			input:      []byte(``),
			wantCost:   0,
			wantTokens: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, tokens, err := parseClaudeOutput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseClaudeOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if cost != tt.wantCost {
				t.Errorf("parseClaudeOutput() cost = %v, want %v", cost, tt.wantCost)
			}
			if tokens != tt.wantTokens {
				t.Errorf("parseClaudeOutput() tokens = %v, want %v", tokens, tt.wantTokens)
			}
		})
	}
}

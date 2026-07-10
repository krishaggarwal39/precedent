package types

// Task represents a single benchmark task derived from a git commit.
// It follows the SWE-bench fail-to-pass task schema.
type Task struct {
	InstanceID       string `json:"instance_id"`
	BaseCommit       string `json:"base_commit"`
	Patch            string `json:"patch"`
	TestPatch        string `json:"test_patch"`
	ProblemStatement string `json:"problem_statement"`
	HintsText        string `json:"hints_text"`
	CreatedAt        string `json:"created_at"`
	Repo             string `json:"repo"`
	Version          string `json:"version"`
	FailToPass       string `json:"FAIL_TO_PASS"`
	PassToPass       string `json:"PASS_TO_PASS"`
	EnvironmentSetup string `json:"environment_setup_commit"`
}

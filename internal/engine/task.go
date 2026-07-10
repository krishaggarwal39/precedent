package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/precedent-cli/precedent/internal/types"
)

// LoadTasks parses all JSON tasks from the given directory.
func LoadTasks(dir string) ([]types.Task, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no tasks found. Run 'precedent init' first")
		}
		return nil, err
	}

	var tasks []types.Task
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			b, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️ Warning: Failed to read %s\n", f.Name())
				continue
			}
			var task types.Task
			if err := json.Unmarshal(b, &task); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️ Warning: Failed to parse %s\n", f.Name())
				continue
			}
			if task.InstanceID == "" || task.BaseCommit == "" {
				fmt.Fprintf(os.Stderr, "⚠️ Warning: Task in %s is missing required fields\n", f.Name())
				continue
			}
			tasks = append(tasks, task)
		}
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no valid tasks found in %s", dir)
	}
	return tasks, nil
}

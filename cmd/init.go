package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/precedent-cli/precedent/internal/miner"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Precedent in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Scanning git history for test-driven fixes...")

		tasksDir := ".precedent/tasks"
		if err := os.MkdirAll(tasksDir, 0755); err != nil {
			return fmt.Errorf("doing mkdir: %w", err)
		}

		repoPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting wd: %w", err)
		}

		// Initialize miner (limit to 10 tasks for now)
		taskMiner := miner.NewTaskMiner(repoPath, 10)
		tasks, err := taskMiner.Mine(context.Background())
		if err != nil {
			return fmt.Errorf("mining tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("❌ No valid tasks found in git history.")
			fmt.Println("Make sure you have commits that modify both source and test files.")
			return nil
		}

		for _, task := range tasks {
			b, err := json.MarshalIndent(task, "", "  ")
			if err != nil {
				return fmt.Errorf("doing marshal: %w", err)
			}

			taskFile := filepath.Join(tasksDir, task.InstanceID+".json")
			if err := os.WriteFile(taskFile, b, 0644); err != nil {
				return fmt.Errorf("doing write task: %w", err)
			}
		}

		fmt.Printf("✅ Generated %d valid tasks in %s\n", len(tasks), tasksDir)
		fmt.Println("Run 'precedent run' to start benchmarking!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

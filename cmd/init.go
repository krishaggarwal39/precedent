package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/precedent-cli/precedent/internal/miner"
	"github.com/precedent-cli/precedent/internal/paths"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Precedent in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting wd: %w", err)
		}

		tasksDir := paths.TasksDir
		// 🚨 Security Hardening: Ensure .precedent is gitignored
		gitignorePath := filepath.Join(repoPath, ".gitignore")
		gitignoreContent, err := os.ReadFile(gitignorePath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reading .gitignore: %w", err)
		}
		if !strings.Contains(string(gitignoreContent), ".precedent/") {
			f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				_, _ = f.WriteString("\n# Precedent CLI Scratch Directory\n.precedent/\n")
				f.Close()
				fmt.Println("🔒 Secured workspace (added .precedent/ to .gitignore)")
			}
		}

		fmt.Println("🔍 Scanning git history for test-driven fixes...")
		if err := os.MkdirAll(tasksDir, 0755); err != nil {
			return fmt.Errorf("doing mkdir: %w", err)
		}

		relaxed, _ := cmd.Flags().GetBool("relaxed")
		testCmd, _ := cmd.Flags().GetString("test-cmd")

		// 🚨 Smart Detection Logic
		if testCmd == "" && !relaxed {
			fmt.Println("🔍 Auto-detecting test framework...")
			detectedCmd := miner.DetectTestCommand(repoPath)
			if detectedCmd != "" {
				testCmd = detectedCmd
				fmt.Printf("✅ Detected test command: '%s'\n", testCmd)
			} else {
				fmt.Println("⚠️  Warning: Could not auto-detect test framework. Falling back to Relaxed Mode (skipping strict fail-to-pass validation). Provide --test-cmd to enable strict mode.")
				relaxed = true
			}
		}

		maxTasks, _ := cmd.Flags().GetInt("max-tasks")
		taskTimeout, _ := cmd.Flags().GetDuration("task-timeout")
		
		// Initialize miner
		taskMiner := miner.NewTaskMiner(repoPath, maxTasks, testCmd, relaxed, taskTimeout)
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
	initCmd.Flags().String("test-cmd", "", "Command to run tests (e.g. 'npm test', 'pytest')")
	initCmd.Flags().Bool("relaxed", false, "Skip strict fail-to-pass test validation")
	initCmd.Flags().Int("max-tasks", 10, "Maximum number of tasks to mine (<= 0 for unlimited)")
	initCmd.Flags().Duration("task-timeout", 10*time.Minute, "Timeout per task validation (prevent infinite test hangs)")
	rootCmd.AddCommand(initCmd)
}

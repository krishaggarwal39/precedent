package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/types"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the benchmark on generated tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		tasksDir := ".precedent/tasks"
		
		files, err := os.ReadDir(tasksDir)
		if err != nil {
			return fmt.Errorf("doing read tasks dir (did you run 'precedent init'?): %w", err)
		}

		var tasks []types.Task
		for _, f := range files {
			if filepath.Ext(f.Name()) == ".json" {
				b, err := os.ReadFile(filepath.Join(tasksDir, f.Name()))
				if err != nil {
					return fmt.Errorf("doing read file %s: %w", f.Name(), err)
				}
				var task types.Task
				if err := json.Unmarshal(b, &task); err != nil {
					return fmt.Errorf("doing unmarshal file %s: %w", f.Name(), err)
				}
				tasks = append(tasks, task)
			}
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found. Run 'precedent init' first.")
			return nil
		}

		fmt.Printf("🚀 Running benchmark on %d task(s)...\n", len(tasks))

		// Dummy adapter for phase 1
		agent := &adapters.DummyAdapter{}
		
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		
		g, ctx := errgroup.WithContext(ctx)

		for _, task := range tasks {
			t := task // capture loop var
			g.Go(func() error {
				result, err := agent.Run(ctx, ".", t.ProblemStatement, t)
				if err != nil {
					return fmt.Errorf("task %s failed: %w", t.InstanceID, err)
				}
				fmt.Printf("✅ Task %s finished. Cost: $%.2f, Tokens: %d, Time: %v\n", 
					t.InstanceID, result.CostUSD, result.TotalTokens, result.Duration)
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return fmt.Errorf("doing benchmark run: %w", err)
		}

		fmt.Println("🎉 Benchmark complete!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

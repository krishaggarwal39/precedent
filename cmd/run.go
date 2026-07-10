package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/config"
	"github.com/precedent-cli/precedent/internal/report"
	"github.com/precedent-cli/precedent/internal/runner"
	"github.com/precedent-cli/precedent/internal/types"
)

// UI styles
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#38bdf8")).MarginBottom(1)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#34d399"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#fb7185"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8"))
	boxStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("#334155"))
)

type taskStartMsg struct{ id string }
type taskDoneMsg struct {
	id     string
	result *adapters.AgentResult
}

type runnerModel struct {
	tasks     []types.Task
	results   []*adapters.AgentResult
	spinner   spinner.Model
	statuses  map[string]string
	mu        *sync.Mutex
	doneCount int
}

func initialModel(tasks []types.Task) runnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#818cf8"))

	statuses := make(map[string]string)
	for _, t := range tasks {
		statuses[t.InstanceID] = "pending"
	}

	return runnerModel{
		tasks:    tasks,
		results:  make([]*adapters.AgentResult, len(tasks)),
		spinner:  s,
		statuses: statuses,
		mu:       &sync.Mutex{},
	}
}

func (m runnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m runnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case taskStartMsg:
		m.mu.Lock()
		m.statuses[msg.id] = "running"
		m.mu.Unlock()
		return m, nil
	case taskDoneMsg:
		m.mu.Lock()
		m.statuses[msg.id] = "done"
		m.doneCount++
		for i, t := range m.tasks {
			if t.InstanceID == msg.id {
				m.results[i] = msg.result
				break
			}
		}
		m.mu.Unlock()
		if m.doneCount == len(m.tasks) {
			return m, tea.Quit
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m runnerModel) View() string {
	s := titleStyle.Render("🚀 Precedent Benchmark Runner") + "\n\n"

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, t := range m.tasks {
		status := m.statuses[t.InstanceID]
		if status == "pending" {
			s += pendingStyle.Render(fmt.Sprintf("○ %s", t.InstanceID)) + "\n"
		} else if status == "running" {
			s += fmt.Sprintf("%s %s\n", m.spinner.View(), lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0")).Render(t.InstanceID))
		} else if status == "done" {
			s += successStyle.Render(fmt.Sprintf("✓ %s", t.InstanceID)) + "\n"
		}
	}

	return boxStyle.Render(s)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the benchmark on generated tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		tasksDir := ".precedent/tasks"
		worktreesDir := ".precedent/worktrees"

		if err := os.MkdirAll(worktreesDir, 0755); err != nil {
			return err
		}

		files, err := os.ReadDir(tasksDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println(errorStyle.Render("❌ No tasks found. Run 'precedent init' first."))
				return nil
			}
			return err
		}

		var tasks []types.Task
		for _, f := range files {
			if filepath.Ext(f.Name()) == ".json" {
				b, _ := os.ReadFile(filepath.Join(tasksDir, f.Name()))
				var task types.Task
				json.Unmarshal(b, &task)
				tasks = append(tasks, task)
			}
		}

		if len(tasks) == 0 {
			fmt.Println(errorStyle.Render("❌ No tasks found."))
			return nil
		}

		agentFlag, _ := cmd.Flags().GetString("agent")
		dockerImage, _ := cmd.Flags().GetString("docker-image")
		maxCost, _ := cmd.Flags().GetFloat64("max-cost")

		var agent adapters.AgentAdapter

		// Load custom agents from YAML
		agentConfigs, _ := config.LoadAgentsConfig(".precedent/agents.yaml")
		if cfg, exists := agentConfigs[agentFlag]; exists {
			agent = &adapters.YamlAdapter{
				AgentName: agentFlag,
				Config:    cfg,
			}
		} else if agentFlag == "claude" || agentFlag == "claude-code" {
			agent = &adapters.ClaudeAdapter{}
		} else {
			fmt.Printf(errorStyle.Render("❌ Unknown agent: %s. Define it in .precedent/agents.yaml or use 'claude'")+"\n", agentFlag)
			return nil
		}

		p := tea.NewProgram(initialModel(tasks))

		// 🚨 FIX: Handle non-TTY environments gracefully (e.g. CI/CD or background tasks)
		isTTY := false
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			isTTY = true
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cores := runtime.NumCPU()
		requestedConc, _ := cmd.Flags().GetInt("concurrency")
		finalConc := requestedConc
		maxSafe := cores * 2

		if finalConc <= 0 {
			finalConc = cores / 2
			if finalConc < 1 {
				finalConc = 1
			}
		} else if finalConc > maxSafe {
			finalConc = maxSafe
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308")).Render(
				fmt.Sprintf("⚠️  Warning: Requested concurrency %d is too high for %d cores.", requestedConc, cores),
			))
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308")).Render(
				fmt.Sprintf("⚠️  Capping concurrency to %d to prevent system crash.", finalConc),
			))
			time.Sleep(2 * time.Second)
		}

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(finalConc)
		repoPath, _ := os.Getwd()

		var currentCost float64
		var costMu sync.Mutex

		go func() {
			for _, task := range tasks {
				t := task
				g.Go(func() error {
					if ctx.Err() != nil {
						return nil // Global context cancelled due to max cost
					}
					if isTTY {
						p.Send(taskStartMsg{id: t.InstanceID})
					} else {
						fmt.Printf("▶️ Starting %s...\n", t.InstanceID)
					}

					wtDir := filepath.Join(repoPath, worktreesDir, t.InstanceID)
					isolation := runner.NewGitIsolation(repoPath, wtDir, t.BaseCommit)
					_ = isolation.Setup(ctx)
					defer isolation.Teardown(context.Background())

					// 🚨 Security Hardening: Strict 10-minute timeout per task to prevent runaway costs
					agentCtx, agentCancel := context.WithTimeout(ctx, 10*time.Minute)
					defer agentCancel()

					var result *adapters.AgentResult
					if agent.IsInstalled() {
						result, _ = agent.Run(agentCtx, wtDir, t.ProblemStatement, t)
					} else {
						dummy := &adapters.DummyAdapter{}
						result, _ = dummy.Run(agentCtx, wtDir, t.ProblemStatement, t)
					}

					// 🚨 The Final Exam & Sandboxing
					if result.Error == nil && t.TestCommand != "" {
						var testCmd *exec.Cmd
						if dockerImage != "" {
							absWtDir, _ := filepath.Abs(wtDir)
							dockerArgs := []string{"run", "--rm", "-v", fmt.Sprintf("%s:/workspace", absWtDir), "-w", "/workspace", dockerImage, "sh", "-c", t.TestCommand}
							testCmd = exec.CommandContext(agentCtx, "docker", dockerArgs...)
						} else {
							testCmd = exec.CommandContext(agentCtx, "sh", "-c", t.TestCommand)
							testCmd.Dir = wtDir
						}

						out, err := testCmd.CombinedOutput()
						if err != nil {
							result.Error = fmt.Errorf("Test failed: %v\nOutput: %s", err, string(out))
						}
					}

					if isTTY {
						p.Send(taskDoneMsg{id: t.InstanceID, result: result})
					} else {
						fmt.Printf("✅ Finished %s in %v\n", t.InstanceID, result.Duration)
					}

					costMu.Lock()
					currentCost += result.CostUSD
					if maxCost > 0 && currentCost >= maxCost {
						if isTTY {
							// For MVP, just printing over the TUI might be messy, but it's a critical alert
						}
						fmt.Println(errorStyle.Render(fmt.Sprintf("\n🚨 ALERT: Max cost of $%.2f exceeded (Current: $%.2f). Cancelling remaining tasks to protect your budget!\n", maxCost, currentCost)))
						cancel()
					}
					costMu.Unlock()

					return nil
				})
			}
			_ = g.Wait()
			if !isTTY {
				fmt.Println("Fallback non-TTY run complete. Scorecard generation requires TTY in V1.")
				os.Exit(0)
			}
		}()

		if isTTY {
			m, err := p.Run()
			if err != nil {
				return err
			}

			finalModel := m.(runnerModel)

			fmt.Println("\n" + titleStyle.Render("📊 Generating Scorecard..."))

			reportPath := "scorecard.html"
			if err := report.GenerateScorecard(finalModel.tasks, finalModel.results, reportPath); err != nil {
				return err
			}

			successMsg := fmt.Sprintf("✨ Benchmark Complete! Open %s to view your premium results.", reportPath)
			fmt.Println(successStyle.Render(successMsg))
		}
		return nil
	},
}

func init() {
	runCmd.Flags().StringP("agent", "a", "claude", "Agent to run (e.g., claude, or custom agent defined in .precedent/agents.yaml)")
	runCmd.Flags().IntP("concurrency", "c", 0, "Number of concurrent tasks (0 = smart auto-detect based on CPU)")
	runCmd.Flags().String("docker-image", "", "Docker image to run tests in (e.g. node:18). If empty, runs locally.")
	runCmd.Flags().Float64("max-cost", 5.0, "Maximum total cost in USD before aborting (0 for unlimited)")
	rootCmd.AddCommand(runCmd)
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/config"
	"github.com/precedent-cli/precedent/internal/engine"
	"github.com/precedent-cli/precedent/internal/paths"
	"github.com/precedent-cli/precedent/internal/report"
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
		if err := os.MkdirAll(paths.WorktreesDir, 0755); err != nil {
			return err
		}

		tasks, err := engine.LoadTasks(paths.TasksDir)
		if err != nil {
			return err
		}

		agentFlag, _ := cmd.Flags().GetString("agent")
		dockerImage, _ := cmd.Flags().GetString("docker-image")
		maxCost, _ := cmd.Flags().GetFloat64("max-cost")
		testCmd, _ := cmd.Flags().GetString("test-cmd")
		taskTimeout, _ := cmd.Flags().GetDuration("task-timeout")

		// Load custom agents from YAML
		agentConfigs, err := config.LoadAgentsConfig(paths.AgentsConfig)
		if err != nil {
			return err
		}

		agent, err := adapters.Resolve(agentFlag, agentConfigs)
		if err != nil {
			return err
		}

		if !agent.IsInstalled() {
			return fmt.Errorf("Agent '%s' is not installed or available in PATH", agentFlag)
		}

		skipConfirm, _ := cmd.Flags().GetBool("yes")
		if !skipConfirm {
			fmt.Println(titleStyle.Render("🛡️ Security Verification (Trust Boundary)"))
			fmt.Println("You are about to run tests against the agent's code.")

			fmt.Printf("• Agent: %s\n", agent.Name())
			if testCmd == "" {
				fmt.Println("• Test Command: (none — results will be marked UNVERIFIED)")
			} else {
				fmt.Printf("• Test Command: %s\n", testCmd)
			}
			if dockerImage != "" {
				fmt.Printf("• Environment: Docker (%s)\n", dockerImage)
			} else {
				fmt.Println("• Environment: Host")
				fmt.Println("⚠️ Tests will execute directly on this machine.")
			}

			fmt.Print("\nDo you want to proceed? (y/N): ")
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" && response != "yes" {
				fmt.Println(errorStyle.Render("❌ Benchmark aborted by user."))
				return nil
			}
		}

		p := tea.NewProgram(initialModel(tasks))

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

		repoPath, _ := os.Getwd()

		engCfg := engine.Config{
			RepoPath:    repoPath,
			TasksDir:    paths.TasksDir,
			WorktreeDir: paths.WorktreesDir,
			MaxCost:     maxCost,
			Concurrency: finalConc,
			DockerImage: dockerImage,
			TaskTimeout: taskTimeout,
			TestCommand: testCmd,
		}

		eng := engine.New(engCfg, agent)
		events, results, err := eng.Run(ctx, tasks)
		if err != nil {
			return err
		}

		// Event loop
		eventDone := make(chan struct{})
		go func() {
			defer close(eventDone)
			for event := range events {
				switch event.Type {
				case engine.EventTaskStarted:
					if isTTY {
						go p.Send(taskStartMsg{id: event.InstanceID})
					} else {
						fmt.Printf("▶️ Starting %s...\n", event.InstanceID)
					}
				case engine.EventTaskFinished:
					if isTTY {
						go p.Send(taskDoneMsg{id: event.InstanceID, result: event.Result})
					} else {
						fmt.Printf("✅ Finished %s in %v\n", event.InstanceID, event.Result.Duration)
					}
				case engine.EventBudgetExceeded:
					if !isTTY {
						fmt.Println(errorStyle.Render(fmt.Sprintf("\n🚨 ALERT: %s\n", event.Message)))
					}
				}
			}
			if isTTY {
				go p.Send(tea.QuitMsg{})
			}
		}()

		if isTTY {
			_, err := p.Run()
			cancel() // User pressed 'q', stop running tasks
			if err != nil {
				<-eventDone
				return err
			}
		}

		<-eventDone

		fmt.Println("\n" + titleStyle.Render("📊 Generating Scorecard..."))

		reportPath := "scorecard.html"
		if err := report.GenerateScorecard(tasks, results, reportPath); err != nil {
			return err
		}

		successMsg := fmt.Sprintf("✨ Benchmark Complete! Open %s to view your premium results.", reportPath)
		fmt.Println(successStyle.Render(successMsg))
		return nil
	},
}

func init() {
	runCmd.Flags().StringP("agent", "a", "claude", "Agent to run (e.g., claude, or custom agent defined in .precedent/agents.yaml)")
	runCmd.Flags().IntP("concurrency", "c", 0, "Number of concurrent tasks (0 = smart auto-detect based on CPU)")
	runCmd.Flags().String("docker-image", "", "Docker image to run tests in (e.g. node:18). If empty, runs locally.")
	runCmd.Flags().Float64("max-cost", 5.0, "Maximum total cost in USD before aborting (0 for unlimited). Currently supports claude adapter only; YAML agents report $0.")
	runCmd.Flags().BoolP("yes", "y", false, "Skip interactive confirmation for test commands")
	runCmd.Flags().String("test-cmd", "", "The test command to execute to verify correctness")
	runCmd.Flags().Duration("task-timeout", 10*time.Minute, "Timeout per task")
	rootCmd.AddCommand(runCmd)
}

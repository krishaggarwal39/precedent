package report

import (
	"html/template"
	"os"
	"time"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/types"
)

type ScorecardData struct {
	TotalTasks    int
	TotalCost     float64
	TotalTime     string
	AverageTokens int
	Results       []TaskResult
	Date          string
}

type TaskResult struct {
	Task        types.Task
	Result      *adapters.AgentResult
	Status      string
	StatusColor string
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Precedent Benchmark Scorecard</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
        body {
            font-family: 'Inter', sans-serif;
            background-color: #0f172a;
            color: #f8fafc;
        }
        .glass {
            background: rgba(30, 41, 59, 0.7);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.1);
        }
        .glow {
            box-shadow: 0 0 20px rgba(56, 189, 248, 0.15);
        }
    </style>
</head>
<body class="min-h-screen p-8 bg-[url('https://www.transparenttextures.com/patterns/cubes.png')]">
    <div class="max-w-6xl mx-auto animate-fade-in-up">
        <!-- Header -->
        <header class="flex justify-between items-center mb-10">
            <div>
                <h1 class="text-4xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-sky-400 to-indigo-500 tracking-tight">Precedent Scorecard</h1>
                <p class="text-slate-400 mt-2 text-sm">Generated on {{ .Date }}</p>
            </div>
            <div class="glass px-4 py-2 rounded-full border border-slate-700/50 flex items-center gap-2">
                <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                <span class="text-sm text-slate-300 font-medium">Benchmark Complete</span>
            </div>
        </header>

        <!-- KPI Cards -->
        <div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-12">
            <div class="glass glow rounded-2xl p-6 relative overflow-hidden group hover:-translate-y-1 transition-all duration-300 cursor-default">
                <div class="absolute inset-0 bg-gradient-to-br from-indigo-500/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <h3 class="text-slate-400 text-sm font-medium mb-2">Total Tasks</h3>
                <p class="text-3xl font-bold text-white">{{ .TotalTasks }}</p>
            </div>
            <div class="glass glow rounded-2xl p-6 relative overflow-hidden group hover:-translate-y-1 transition-all duration-300 cursor-default">
                <div class="absolute inset-0 bg-gradient-to-br from-emerald-500/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <h3 class="text-slate-400 text-sm font-medium mb-2">Total Cost</h3>
                <p class="text-3xl font-bold text-emerald-400">${{ printf "%.2f" .TotalCost }}</p>
            </div>
            <div class="glass glow rounded-2xl p-6 relative overflow-hidden group hover:-translate-y-1 transition-all duration-300 cursor-default">
                <div class="absolute inset-0 bg-gradient-to-br from-amber-500/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <h3 class="text-slate-400 text-sm font-medium mb-2">Avg Tokens / Task</h3>
                <p class="text-3xl font-bold text-amber-400">{{ .AverageTokens }}</p>
            </div>
            <div class="glass glow rounded-2xl p-6 relative overflow-hidden group hover:-translate-y-1 transition-all duration-300 cursor-default">
                <div class="absolute inset-0 bg-gradient-to-br from-rose-500/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
                <h3 class="text-slate-400 text-sm font-medium mb-2">Total Time</h3>
                <p class="text-3xl font-bold text-rose-400">{{ .TotalTime }}</p>
            </div>
        </div>

        <!-- Table -->
        <div class="glass rounded-2xl overflow-hidden glow mb-12">
            <table class="w-full text-left border-collapse">
                <thead>
                    <tr class="bg-slate-800/50 text-slate-300 border-b border-slate-700/50 text-sm">
                        <th class="p-4 font-semibold">Task ID</th>
                        <th class="p-4 font-semibold">Problem Statement</th>
                        <th class="p-4 font-semibold text-right">Cost</th>
                        <th class="p-4 font-semibold text-right">Time</th>
                        <th class="p-4 font-semibold text-center">Status</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-slate-700/50 text-sm">
                    {{ range .Results }}
                    <tr class="hover:bg-slate-800/30 transition-colors">
                        <td class="p-4 font-mono text-xs text-sky-300">{{ .Task.InstanceID }}</td>
                        <td class="p-4 text-slate-300 max-w-xl truncate" title="{{ .Task.ProblemStatement }}">{{ .Task.ProblemStatement }}</td>
                        <td class="p-4 text-right font-medium">${{ printf "%.2f" .Result.CostUSD }}</td>
                        <td class="p-4 text-right text-slate-400">{{ .Result.Duration }}</td>
                        <td class="p-4 text-center">
                            <span class="px-3 py-1 rounded-full text-xs font-medium {{ .StatusColor }} bg-opacity-10 border border-current shadow-sm">
                                {{ .Status }}
                            </span>
                        </td>
                    </tr>
                    {{ end }}
                </tbody>
            </table>
        </div>

        <!-- Limitations Footer -->
        <div class="glass rounded-2xl p-6 text-sm text-slate-400 border border-slate-700/50">
            <h4 class="text-slate-300 font-semibold mb-2">⚠️ Benchmark Limitations</h4>
            <ul class="list-disc list-inside space-y-1">
                <li><strong>Pass@1 Only:</strong> Agents are given a single attempt without retry feedback.</li>
                <li><strong>Non-Deterministic:</strong> AI outputs vary. A PASS today may be a FAIL tomorrow due to random seed variance.</li>
                <li><strong>Flaky Tests:</strong> While Fail-to-Pass validation is enforced, environmental race conditions may occasionally cause false negatives.</li>
            </ul>
        </div>
    </div>
</body>
</html>`

// GenerateScorecard compiles the benchmark results into a beautiful HTML file.
func GenerateScorecard(tasks []types.Task, results []*adapters.AgentResult, outPath string) error {
	var totalCost float64
	var totalTokens int
	var totalDuration time.Duration

	var tableResults []TaskResult

	for i, task := range tasks {
		res := results[i]
		totalCost += res.CostUSD
		totalTokens += res.TotalTokens
		totalDuration += res.Duration

		status := "PASS"
		statusColor := "text-emerald-400 border-emerald-400 bg-emerald-400"
		if res.Error != nil {
			status = "FAIL"
			statusColor = "text-rose-400 border-rose-400 bg-rose-400"
		}

		tableResults = append(tableResults, TaskResult{
			Task:        task,
			Result:      res,
			Status:      status,
			StatusColor: statusColor,
		})
	}

	avgTokens := 0
	if len(tasks) > 0 {
		avgTokens = totalTokens / len(tasks)
	}

	data := ScorecardData{
		TotalTasks:    len(tasks),
		TotalCost:     totalCost,
		TotalTime:     totalDuration.String(),
		AverageTokens: avgTokens,
		Results:       tableResults,
		Date:          time.Now().Format("02 Jan 2006 15:04"),
	}

	tmpl, err := template.New("scorecard").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

package report

import (
	"html/template"
	"os"
	"time"

	"github.com/precedent-cli/precedent/internal/adapters"
	"github.com/precedent-cli/precedent/internal/types"
)

// ScorecardData holds the aggregated data required to render the HTML scorecard.
type ScorecardData struct {
	TotalTasks    int
	TotalCost     float64
	TotalTime     string
	AverageTokens int
	Results       []TaskResult
	Date          string
}

// TaskResult pairs a benchmark task with its execution result and derived status.
type TaskResult struct {
	Task   types.Task
	Result *adapters.AgentResult
	Status string
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Precedent Benchmark Scorecard</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            background-color: #0f172a;
            color: #f8fafc;
            margin: 0;
            padding: 2rem;
            min-height: 100vh;
        }
        .container {
            max-width: 72rem;
            margin: 0 auto;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2.5rem;
        }
        .title {
            font-size: 2.25rem;
            font-weight: 700;
            margin: 0;
            background: linear-gradient(to right, #38bdf8, #6366f1);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -0.025em;
        }
        .subtitle {
            color: #94a3b8;
            margin-top: 0.5rem;
            font-size: 0.875rem;
        }
        .glass {
            background: rgba(30, 41, 59, 0.7);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.1);
        }
        .status-badge {
            padding: 0.5rem 1rem;
            border-radius: 9999px;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.875rem;
            font-weight: 500;
            color: #cbd5e1;
        }
        .pulse {
            width: 0.5rem;
            height: 0.5rem;
            border-radius: 9999px;
            background-color: #34d399;
            animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: .5; }
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(1, 1fr);
            gap: 1.5rem;
            margin-bottom: 3rem;
        }
        @media (min-width: 768px) {
            .grid { grid-template-columns: repeat(4, 1fr); }
        }
        .card {
            border-radius: 1rem;
            padding: 1.5rem;
            box-shadow: 0 0 20px rgba(56, 189, 248, 0.15);
        }
        .card-title {
            color: #94a3b8;
            font-size: 0.875rem;
            font-weight: 500;
            margin: 0 0 0.5rem 0;
        }
        .card-value {
            font-size: 1.875rem;
            font-weight: 700;
            margin: 0;
        }
        .text-emerald { color: #34d399; }
        .text-amber { color: #fbbf24; }
        .text-rose { color: #fb7185; }
        
        .table-container {
            border-radius: 1rem;
            overflow: hidden;
            margin-bottom: 3rem;
            box-shadow: 0 0 20px rgba(56, 189, 248, 0.15);
        }
        table {
            width: 100%;
            border-collapse: collapse;
            text-align: left;
        }
        th {
            background-color: rgba(30, 41, 59, 0.5);
            color: #cbd5e1;
            padding: 1rem;
            font-weight: 600;
            font-size: 0.875rem;
            border-bottom: 1px solid rgba(51, 65, 85, 0.5);
        }
        td {
            padding: 1rem;
            font-size: 0.875rem;
            color: #cbd5e1;
            border-bottom: 1px solid rgba(51, 65, 85, 0.5);
        }
        .text-right { text-align: right; }
        .text-center { text-align: center; }
        .font-mono { font-family: ui-monospace, monospace; color: #7dd3fc; font-size: 0.75rem; }
        .truncate {
            max-width: 32rem;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }
        
        .pill {
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 500;
            border: 1px solid currentColor;
            text-transform: uppercase;
        }
        .status-pass { color: #34d399; background-color: rgba(52, 211, 153, 0.1); }
        .status-fail { color: #fb7185; background-color: rgba(251, 113, 133, 0.1); }
        .status-unverified { color: #fbbf24; background-color: rgba(251, 191, 36, 0.1); }
        .status-skipped { color: #94a3b8; background-color: rgba(148, 163, 184, 0.1); }
        
        .footer {
            border-radius: 1rem;
            padding: 1.5rem;
            color: #94a3b8;
            font-size: 0.875rem;
            border: 1px solid rgba(255, 255, 255, 0.1);
        }
        .footer h4 {
            color: #cbd5e1;
            font-weight: 600;
            margin: 0 0 0.5rem 0;
        }
        .footer ul {
            margin: 0;
            padding-left: 1.5rem;
        }
        .footer li {
            margin-bottom: 0.25rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Header -->
        <div class="header">
            <div>
                <h1 class="title">Precedent Scorecard</h1>
                <p class="subtitle">Generated on {{ .Date }}</p>
            </div>
            <div class="glass status-badge">
                <span class="pulse"></span>
                Benchmark Complete
            </div>
        </div>

        <!-- KPI Cards -->
        <div class="grid">
            <div class="glass card">
                <h3 class="card-title">Total Tasks</h3>
                <p class="card-value">{{ .TotalTasks }}</p>
            </div>
            <div class="glass card">
                <h3 class="card-title">Total Cost</h3>
                <p class="card-value text-emerald">${{ printf "%.2f" .TotalCost }}</p>
            </div>
            <div class="glass card">
                <h3 class="card-title">Avg Tokens / Task</h3>
                <p class="card-value text-amber">{{ .AverageTokens }}</p>
            </div>
            <div class="glass card">
                <h3 class="card-title">Total Time</h3>
                <p class="card-value text-rose">{{ .TotalTime }}</p>
            </div>
        </div>

        <!-- Table -->
        <div class="glass table-container">
            <table>
                <thead>
                    <tr>
                        <th>Task ID</th>
                        <th>Problem Statement</th>
                        <th class="text-right">Cost</th>
                        <th class="text-right">Time</th>
                        <th class="text-center">Status</th>
                    </tr>
                </thead>
                <tbody>
                    {{ range .Results }}
                    <tr>
                        <td class="font-mono">{{ .Task.InstanceID }}</td>
                        <td><div class="truncate" title="{{ .Task.ProblemStatement }}">{{ .Task.ProblemStatement }}</div></td>
                        <td class="text-right">${{ printf "%.2f" .Result.CostUSD }}</td>
                        <td class="text-right">{{ .Result.Duration }}</td>
                        <td class="text-center">
                            <span class="pill status-{{ .Status }}">
                                {{ .Status }}
                            </span>
                        </td>
                    </tr>
                    {{ end }}
                </tbody>
            </table>
        </div>

        <!-- Limitations Footer -->
        <div class="glass footer">
            <h4>⚠️ Benchmark Limitations</h4>
            <ul>
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

		var status string
		if res == nil {
			status = "skipped"
			res = &adapters.AgentResult{Duration: 0} // Prevent template nil panics
		} else {
			totalCost += res.CostUSD
			totalTokens += res.TotalTokens
			totalDuration += res.Duration

			if res.Error != nil {
				status = "fail"
			} else if res.Unverified {
				status = "unverified"
			} else {
				status = "pass"
			}
		}

		tableResults = append(tableResults, TaskResult{
			Task:   task,
			Result: res,
			Status: status,
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

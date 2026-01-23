// genreport generates HTML reports from Cucumber JSON output
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"
)

// Cucumber JSON structures
type CucumberReport []Feature

type Feature struct {
	URI         string     `json:"uri"`
	ID          string     `json:"id"`
	Keyword     string     `json:"keyword"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Line        int        `json:"line"`
	Tags        []Tag      `json:"tags"`
	Elements    []Scenario `json:"elements"`
}

type Scenario struct {
	ID          string `json:"id"`
	Keyword     string `json:"keyword"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Line        int    `json:"line"`
	Type        string `json:"type"`
	Tags        []Tag  `json:"tags"`
	Steps       []Step `json:"steps"`
}

type Step struct {
	Keyword string `json:"keyword"`
	Name    string `json:"name"`
	Line    int    `json:"line"`
	Result  Result `json:"result"`
}

type Result struct {
	Status   string `json:"status"`
	Duration int64  `json:"duration"`
	Error    string `json:"error_message,omitempty"`
}

type Tag struct {
	Name string `json:"name"`
	Line int    `json:"line"`
}

// Report data for template
type ReportData struct {
	Title         string
	GeneratedAt   string
	TotalFeatures int
	TotalScenarios int
	TotalSteps    int
	PassedScenarios int
	FailedScenarios int
	SkippedScenarios int
	PassedSteps   int
	FailedSteps   int
	SkippedSteps  int
	Features      []FeatureReport
}

type FeatureReport struct {
	Name        string
	URI         string
	Tags        string
	Scenarios   []ScenarioReport
	PassCount   int
	FailCount   int
	SkipCount   int
}

type ScenarioReport struct {
	Name   string
	Tags   string
	Status string
	Steps  []StepReport
}

type StepReport struct {
	Keyword  string
	Name     string
	Status   string
	Duration string
	Error    string
}

func main() {
	inputFile := flag.String("input", "cucumber.json", "Input Cucumber JSON file")
	outputFile := flag.String("output", "report.html", "Output HTML file")
	title := flag.String("title", "Backlog CLI - Specification Report", "Report title")
	flag.Parse()

	// Read input JSON
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	var report CucumberReport
	if err := json.Unmarshal(data, &report); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Transform to report data
	reportData := transformReport(report, *title)

	// Generate HTML
	if err := generateHTML(reportData, *outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("HTML report generated: %s\n", *outputFile)
	fmt.Printf("Features: %d, Scenarios: %d (passed: %d, failed: %d, skipped: %d)\n",
		reportData.TotalFeatures, reportData.TotalScenarios,
		reportData.PassedScenarios, reportData.FailedScenarios, reportData.SkippedScenarios)
}

func transformReport(report CucumberReport, title string) ReportData {
	data := ReportData{
		Title:       title,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	for _, feature := range report {
		fr := FeatureReport{
			Name: feature.Name,
			URI:  feature.URI,
			Tags: formatTags(feature.Tags),
		}

		for _, scenario := range feature.Elements {
			if scenario.Type == "background" {
				continue // Skip background sections
			}

			sr := ScenarioReport{
				Name:   scenario.Name,
				Tags:   formatTags(scenario.Tags),
				Status: "passed",
			}

			for _, step := range scenario.Steps {
				stepStatus := step.Result.Status
				if stepStatus == "" {
					stepStatus = "skipped"
				}

				str := StepReport{
					Keyword:  step.Keyword,
					Name:     step.Name,
					Status:   stepStatus,
					Duration: formatDuration(step.Result.Duration),
					Error:    step.Result.Error,
				}
				sr.Steps = append(sr.Steps, str)

				data.TotalSteps++
				switch stepStatus {
				case "passed":
					data.PassedSteps++
				case "failed":
					data.FailedSteps++
					sr.Status = "failed"
				default:
					data.SkippedSteps++
					if sr.Status == "passed" {
						sr.Status = "skipped"
					}
				}
			}

			fr.Scenarios = append(fr.Scenarios, sr)
			data.TotalScenarios++

			switch sr.Status {
			case "passed":
				data.PassedScenarios++
				fr.PassCount++
			case "failed":
				data.FailedScenarios++
				fr.FailCount++
			default:
				data.SkippedScenarios++
				fr.SkipCount++
			}
		}

		data.Features = append(data.Features, fr)
		data.TotalFeatures++
	}

	return data
}

func formatTags(tags []Tag) string {
	if len(tags) == 0 {
		return ""
	}
	var names []string
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return strings.Join(names, " ")
}

func formatDuration(ns int64) string {
	if ns == 0 {
		return ""
	}
	d := time.Duration(ns)
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func generateHTML(data ReportData, outputFile string) error {
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        :root {
            --color-passed: #22c55e;
            --color-failed: #ef4444;
            --color-skipped: #f59e0b;
            --color-bg: #0f172a;
            --color-surface: #1e293b;
            --color-border: #334155;
            --color-text: #e2e8f0;
            --color-text-muted: #94a3b8;
        }
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: var(--color-bg);
            color: var(--color-text);
            line-height: 1.6;
            padding: 2rem;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        header {
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--color-border);
        }
        h1 {
            font-size: 1.875rem;
            font-weight: 600;
            margin-bottom: 0.5rem;
        }
        .generated {
            color: var(--color-text-muted);
            font-size: 0.875rem;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        .stat {
            background: var(--color-surface);
            padding: 1rem;
            border-radius: 0.5rem;
            text-align: center;
        }
        .stat-value {
            font-size: 2rem;
            font-weight: 700;
        }
        .stat-label {
            color: var(--color-text-muted);
            font-size: 0.875rem;
        }
        .stat-passed .stat-value { color: var(--color-passed); }
        .stat-failed .stat-value { color: var(--color-failed); }
        .stat-skipped .stat-value { color: var(--color-skipped); }
        .feature {
            background: var(--color-surface);
            border-radius: 0.5rem;
            margin-bottom: 1rem;
            overflow: hidden;
        }
        .feature-header {
            padding: 1rem;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid var(--color-border);
        }
        .feature-header:hover {
            background: rgba(255,255,255,0.05);
        }
        .feature-name {
            font-weight: 600;
        }
        .feature-uri {
            color: var(--color-text-muted);
            font-size: 0.75rem;
            margin-top: 0.25rem;
        }
        .feature-stats {
            display: flex;
            gap: 0.75rem;
            font-size: 0.875rem;
        }
        .badge {
            padding: 0.125rem 0.5rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            font-weight: 500;
        }
        .badge-passed { background: rgba(34, 197, 94, 0.2); color: var(--color-passed); }
        .badge-failed { background: rgba(239, 68, 68, 0.2); color: var(--color-failed); }
        .badge-skipped { background: rgba(245, 158, 11, 0.2); color: var(--color-skipped); }
        .scenarios {
            padding: 0 1rem 1rem;
        }
        .scenario {
            border: 1px solid var(--color-border);
            border-radius: 0.375rem;
            margin-top: 0.75rem;
            overflow: hidden;
        }
        .scenario-header {
            padding: 0.75rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            cursor: pointer;
            background: rgba(0,0,0,0.2);
        }
        .scenario-header:hover {
            background: rgba(0,0,0,0.3);
        }
        .scenario-name {
            font-weight: 500;
        }
        .scenario-tags {
            color: var(--color-text-muted);
            font-size: 0.75rem;
            margin-left: 0.5rem;
        }
        .steps {
            padding: 0.5rem;
            display: none;
        }
        .scenario.expanded .steps {
            display: block;
        }
        .step {
            padding: 0.5rem;
            font-family: 'SF Mono', Monaco, Consolas, monospace;
            font-size: 0.8125rem;
            border-radius: 0.25rem;
            margin-bottom: 0.25rem;
        }
        .step:last-child {
            margin-bottom: 0;
        }
        .step-passed { background: rgba(34, 197, 94, 0.1); border-left: 3px solid var(--color-passed); }
        .step-failed { background: rgba(239, 68, 68, 0.1); border-left: 3px solid var(--color-failed); }
        .step-skipped { background: rgba(245, 158, 11, 0.1); border-left: 3px solid var(--color-skipped); }
        .step-keyword {
            color: #818cf8;
            font-weight: 600;
        }
        .step-duration {
            color: var(--color-text-muted);
            font-size: 0.75rem;
            margin-left: 0.5rem;
        }
        .step-error {
            color: var(--color-failed);
            font-size: 0.75rem;
            margin-top: 0.5rem;
            padding: 0.5rem;
            background: rgba(239, 68, 68, 0.1);
            border-radius: 0.25rem;
            white-space: pre-wrap;
            word-break: break-word;
        }
        .tags {
            color: #a78bfa;
            font-size: 0.75rem;
        }
        .toggle-all {
            background: var(--color-surface);
            border: 1px solid var(--color-border);
            color: var(--color-text);
            padding: 0.5rem 1rem;
            border-radius: 0.375rem;
            cursor: pointer;
            margin-bottom: 1rem;
        }
        .toggle-all:hover {
            background: rgba(255,255,255,0.1);
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>{{.Title}}</h1>
            <p class="generated">Generated: {{.GeneratedAt}}</p>
        </header>

        <div class="summary">
            <div class="stat">
                <div class="stat-value">{{.TotalFeatures}}</div>
                <div class="stat-label">Features</div>
            </div>
            <div class="stat">
                <div class="stat-value">{{.TotalScenarios}}</div>
                <div class="stat-label">Scenarios</div>
            </div>
            <div class="stat stat-passed">
                <div class="stat-value">{{.PassedScenarios}}</div>
                <div class="stat-label">Passed</div>
            </div>
            <div class="stat stat-failed">
                <div class="stat-value">{{.FailedScenarios}}</div>
                <div class="stat-label">Failed</div>
            </div>
            <div class="stat stat-skipped">
                <div class="stat-value">{{.SkippedScenarios}}</div>
                <div class="stat-label">Skipped</div>
            </div>
        </div>

        <button class="toggle-all" onclick="toggleAll()">Expand/Collapse All</button>

        {{range .Features}}
        <div class="feature">
            <div class="feature-header" onclick="toggleFeature(this)">
                <div>
                    <div class="feature-name">{{.Name}}</div>
                    <div class="feature-uri">{{.URI}}</div>
                    {{if .Tags}}<div class="tags">{{.Tags}}</div>{{end}}
                </div>
                <div class="feature-stats">
                    {{if gt .PassCount 0}}<span class="badge badge-passed">{{.PassCount}} passed</span>{{end}}
                    {{if gt .FailCount 0}}<span class="badge badge-failed">{{.FailCount}} failed</span>{{end}}
                    {{if gt .SkipCount 0}}<span class="badge badge-skipped">{{.SkipCount}} skipped</span>{{end}}
                </div>
            </div>
            <div class="scenarios">
                {{range .Scenarios}}
                <div class="scenario">
                    <div class="scenario-header" onclick="toggleScenario(this)">
                        <div>
                            <span class="scenario-name">{{.Name}}</span>
                            {{if .Tags}}<span class="scenario-tags">{{.Tags}}</span>{{end}}
                        </div>
                        <span class="badge badge-{{.Status}}">{{.Status}}</span>
                    </div>
                    <div class="steps">
                        {{range .Steps}}
                        <div class="step step-{{.Status}}">
                            <span class="step-keyword">{{.Keyword}}</span>{{.Name}}
                            {{if .Duration}}<span class="step-duration">{{.Duration}}</span>{{end}}
                            {{if .Error}}<div class="step-error">{{.Error}}</div>{{end}}
                        </div>
                        {{end}}
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
    </div>

    <script>
        function toggleFeature(header) {
            const scenarios = header.nextElementSibling;
            scenarios.style.display = scenarios.style.display === 'none' ? 'block' : 'none';
        }
        function toggleScenario(header) {
            header.parentElement.classList.toggle('expanded');
        }
        function toggleAll() {
            const scenarios = document.querySelectorAll('.scenarios');
            const allHidden = Array.from(scenarios).every(s => s.style.display === 'none');
            scenarios.forEach(s => s.style.display = allHidden ? 'block' : 'none');
            if (allHidden) {
                document.querySelectorAll('.scenario').forEach(s => s.classList.add('expanded'));
            } else {
                document.querySelectorAll('.scenario').forEach(s => s.classList.remove('expanded'));
            }
        }
    </script>
</body>
</html>
`

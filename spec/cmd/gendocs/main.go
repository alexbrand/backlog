// gendocs generates living documentation from Gherkin feature files
package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Feature represents a parsed Gherkin feature
type Feature struct {
	Name        string
	Description string
	Tags        []string
	Background  *Background
	Scenarios   []Scenario
	FilePath    string
}

// Background represents a feature's background section
type Background struct {
	Name  string
	Steps []Step
}

// Scenario represents a scenario or scenario outline
type Scenario struct {
	Name        string
	Description string
	Tags        []string
	Steps       []Step
	Examples    []ExampleTable
	IsOutline   bool
}

// Step represents a single step in a scenario
type Step struct {
	Keyword   string
	Text      string
	DocString string
	DataTable [][]string
}

// ExampleTable represents examples for a scenario outline
type ExampleTable struct {
	Name    string
	Headers []string
	Rows    [][]string
}

// DocData is the data structure for the HTML template
type DocData struct {
	Title           string
	GeneratedAt     string
	TotalFeatures   int
	TotalScenarios  int
	FeaturesByPhase []PhaseGroup
}

// PhaseGroup groups features by phase/category
type PhaseGroup struct {
	Name     string
	Features []FeatureDoc
}

// FeatureDoc is a feature formatted for documentation
type FeatureDoc struct {
	Name           string
	Description    string
	Tags           string
	FilePath       string
	Background     *BackgroundDoc
	Scenarios      []ScenarioDoc
	ScenarioCount  int
	OutlineCount   int
}

// BackgroundDoc is a background formatted for documentation
type BackgroundDoc struct {
	Steps []StepDoc
}

// ScenarioDoc is a scenario formatted for documentation
type ScenarioDoc struct {
	Name        string
	Description string
	Tags        string
	Steps       []StepDoc
	Examples    []ExampleTableDoc
	IsOutline   bool
}

// StepDoc is a step formatted for documentation
type StepDoc struct {
	Keyword   string
	Text      string
	DocString string
	DataTable [][]string
	HasExtra  bool
}

// ExampleTableDoc is an example table formatted for documentation
type ExampleTableDoc struct {
	Name    string
	Headers []string
	Rows    [][]string
}

func main() {
	featuresDir := flag.String("features", "features", "Directory containing .feature files")
	outputFile := flag.String("output", "docs.html", "Output HTML file")
	title := flag.String("title", "Backlog CLI - Living Documentation", "Documentation title")
	flag.Parse()

	// Find all feature files
	var featureFiles []string
	err := filepath.WalkDir(*featuresDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".feature") {
			featureFiles = append(featureFiles, path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding feature files: %v\n", err)
		os.Exit(1)
	}

	if len(featureFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No feature files found in %s\n", *featuresDir)
		os.Exit(1)
	}

	// Parse all features
	var features []Feature
	for _, path := range featureFiles {
		feature, err := parseFeatureFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", path, err)
			continue
		}
		features = append(features, feature)
	}

	// Generate documentation
	docData := buildDocData(features, *title)
	if err := generateHTML(docData, *outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Living documentation generated: %s\n", *outputFile)
	fmt.Printf("Features: %d, Scenarios: %d\n", docData.TotalFeatures, docData.TotalScenarios)
}

// parseFeatureFile parses a Gherkin feature file
func parseFeatureFile(path string) (Feature, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Feature{}, err
	}

	return parseGherkin(string(content), path)
}

// parseGherkin parses Gherkin content into a Feature struct
func parseGherkin(content string, filePath string) (Feature, error) {
	feature := Feature{FilePath: filepath.Base(filePath)}
	lines := strings.Split(content, "\n")

	var currentSection string
	var currentScenario *Scenario
	var currentStep *Step
	var inDocString bool
	var docStringIndent int
	var inExamples bool
	var currentExamples *ExampleTable

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Handle doc strings
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, "```") {
			if inDocString {
				inDocString = false
				currentStep = nil
			} else {
				inDocString = true
				docStringIndent = len(line) - len(strings.TrimLeft(line, " \t"))
			}
			continue
		}

		if inDocString {
			if currentStep != nil {
				// Remove the indentation from doc string content
				docLine := line
				if len(docLine) > docStringIndent {
					docLine = docLine[docStringIndent:]
				}
				if currentStep.DocString != "" {
					currentStep.DocString += "\n"
				}
				currentStep.DocString += docLine
			}
			continue
		}

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle tags
		if strings.HasPrefix(trimmed, "@") {
			tags := parseTags(trimmed)
			if currentScenario != nil && currentSection == "scenario" {
				// Tags before examples don't apply to scenario
			} else if currentSection == "" || currentSection == "feature" {
				feature.Tags = append(feature.Tags, tags...)
			}
			continue
		}

		// Handle data tables
		if strings.HasPrefix(trimmed, "|") {
			row := parseTableRow(trimmed)
			if inExamples && currentExamples != nil {
				if len(currentExamples.Headers) == 0 {
					currentExamples.Headers = row
				} else {
					currentExamples.Rows = append(currentExamples.Rows, row)
				}
			} else if currentStep != nil {
				currentStep.DataTable = append(currentStep.DataTable, row)
			}
			continue
		}

		// Feature
		if strings.HasPrefix(trimmed, "Feature:") {
			feature.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "Feature:"))
			currentSection = "feature"
			continue
		}

		// Feature description (lines after Feature: before Background/Scenario)
		if currentSection == "feature" && !strings.HasPrefix(trimmed, "Background:") &&
			!strings.HasPrefix(trimmed, "Scenario:") && !strings.HasPrefix(trimmed, "Scenario Outline:") {
			if feature.Description != "" {
				feature.Description += "\n"
			}
			feature.Description += trimmed
			continue
		}

		// Background
		if strings.HasPrefix(trimmed, "Background:") {
			currentSection = "background"
			feature.Background = &Background{
				Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "Background:")),
			}
			currentScenario = nil
			inExamples = false
			continue
		}

		// Scenario or Scenario Outline
		if strings.HasPrefix(trimmed, "Scenario Outline:") || strings.HasPrefix(trimmed, "Scenario:") {
			currentSection = "scenario"
			inExamples = false

			isOutline := strings.HasPrefix(trimmed, "Scenario Outline:")
			name := trimmed
			if isOutline {
				name = strings.TrimSpace(strings.TrimPrefix(trimmed, "Scenario Outline:"))
			} else {
				name = strings.TrimSpace(strings.TrimPrefix(trimmed, "Scenario:"))
			}

			// Look back for tags on the previous non-empty line
			var tags []string
			for j := i - 1; j >= 0; j-- {
				prevTrimmed := strings.TrimSpace(lines[j])
				if prevTrimmed == "" {
					continue
				}
				if strings.HasPrefix(prevTrimmed, "@") {
					tags = parseTags(prevTrimmed)
				}
				break
			}

			scenario := Scenario{
				Name:      name,
				Tags:      tags,
				IsOutline: isOutline,
			}
			feature.Scenarios = append(feature.Scenarios, scenario)
			currentScenario = &feature.Scenarios[len(feature.Scenarios)-1]
			continue
		}

		// Examples
		if strings.HasPrefix(trimmed, "Examples:") {
			inExamples = true
			if currentScenario != nil {
				exampleTable := ExampleTable{
					Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "Examples:")),
				}
				currentScenario.Examples = append(currentScenario.Examples, exampleTable)
				currentExamples = &currentScenario.Examples[len(currentScenario.Examples)-1]
			}
			continue
		}

		// Steps (Given, When, Then, And, But)
		stepKeywords := []string{"Given ", "When ", "Then ", "And ", "But ", "* "}
		for _, keyword := range stepKeywords {
			if strings.HasPrefix(trimmed, keyword) {
				step := Step{
					Keyword: strings.TrimSpace(keyword),
					Text:    strings.TrimPrefix(trimmed, keyword),
				}

				if currentSection == "background" && feature.Background != nil {
					feature.Background.Steps = append(feature.Background.Steps, step)
					currentStep = &feature.Background.Steps[len(feature.Background.Steps)-1]
				} else if currentScenario != nil {
					currentScenario.Steps = append(currentScenario.Steps, step)
					currentStep = &currentScenario.Steps[len(currentScenario.Steps)-1]
				}
				inExamples = false
				break
			}
		}
	}

	return feature, nil
}

func parseTags(line string) []string {
	var tags []string
	parts := strings.Fields(line)
	for _, part := range parts {
		if strings.HasPrefix(part, "@") {
			tags = append(tags, part)
		}
	}
	return tags
}

func parseTableRow(line string) []string {
	// Remove leading/trailing pipes and split
	line = strings.Trim(line, "|")
	cells := strings.Split(line, "|")
	var row []string
	for _, cell := range cells {
		row = append(row, strings.TrimSpace(cell))
	}
	return row
}

func buildDocData(features []Feature, title string) DocData {
	// Group features by category based on file name
	groups := categorizeFeatures(features)

	var totalScenarios int
	var phaseGroups []PhaseGroup

	for _, group := range groups {
		pg := PhaseGroup{Name: group.name}

		for _, f := range group.features {
			fd := FeatureDoc{
				Name:        f.Name,
				Description: strings.TrimSpace(f.Description),
				Tags:        strings.Join(f.Tags, " "),
				FilePath:    f.FilePath,
			}

			if f.Background != nil {
				bd := &BackgroundDoc{}
				for _, s := range f.Background.Steps {
					bd.Steps = append(bd.Steps, StepDoc{
						Keyword:   s.Keyword,
						Text:      s.Text,
						DocString: s.DocString,
						DataTable: s.DataTable,
						HasExtra:  s.DocString != "" || len(s.DataTable) > 0,
					})
				}
				fd.Background = bd
			}

			for _, s := range f.Scenarios {
				sd := ScenarioDoc{
					Name:        s.Name,
					Description: strings.TrimSpace(s.Description),
					Tags:        strings.Join(s.Tags, " "),
					IsOutline:   s.IsOutline,
				}

				for _, step := range s.Steps {
					sd.Steps = append(sd.Steps, StepDoc{
						Keyword:   step.Keyword,
						Text:      step.Text,
						DocString: step.DocString,
						DataTable: step.DataTable,
						HasExtra:  step.DocString != "" || len(step.DataTable) > 0,
					})
				}

				for _, ex := range s.Examples {
					sd.Examples = append(sd.Examples, ExampleTableDoc{
						Name:    ex.Name,
						Headers: ex.Headers,
						Rows:    ex.Rows,
					})
				}

				fd.Scenarios = append(fd.Scenarios, sd)
				totalScenarios++

				if s.IsOutline {
					fd.OutlineCount++
				}
			}
			fd.ScenarioCount = len(fd.Scenarios)

			pg.Features = append(pg.Features, fd)
		}

		phaseGroups = append(phaseGroups, pg)
	}

	return DocData{
		Title:           title,
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05"),
		TotalFeatures:   len(features),
		TotalScenarios:  totalScenarios,
		FeaturesByPhase: phaseGroups,
	}
}

type featureGroup struct {
	name     string
	features []Feature
}

func categorizeFeatures(features []Feature) []featureGroup {
	categories := map[string][]Feature{
		"Core Commands":      {},
		"Output Formats":     {},
		"Configuration":      {},
		"Agent Coordination": {},
		"Git Integration":    {},
		"GitHub Backend":     {},
		"Linear Backend":     {},
		"Cross-Cutting":      {},
	}

	categoryOrder := []string{
		"Core Commands",
		"Output Formats",
		"Configuration",
		"Agent Coordination",
		"Git Integration",
		"GitHub Backend",
		"Linear Backend",
		"Cross-Cutting",
	}

	// Patterns to categorize features
	patterns := []struct {
		pattern  *regexp.Regexp
		category string
	}{
		{regexp.MustCompile(`^(init|add|list|show|move|edit)\.feature$`), "Core Commands"},
		{regexp.MustCompile(`^output_`), "Output Formats"},
		{regexp.MustCompile(`^(config|global_flags|errors)\.feature$`), "Configuration"},
		{regexp.MustCompile(`^(claim|release|next|comment|locking)\.feature$`), "Agent Coordination"},
		{regexp.MustCompile(`^git_`), "Git Integration"},
		{regexp.MustCompile(`^github_`), "GitHub Backend"},
		{regexp.MustCompile(`^linear_`), "Linear Backend"},
		{regexp.MustCompile(`^(multi_backend|agent_workflow)\.feature$`), "Cross-Cutting"},
	}

	for _, f := range features {
		categorized := false
		for _, p := range patterns {
			if p.pattern.MatchString(f.FilePath) {
				categories[p.category] = append(categories[p.category], f)
				categorized = true
				break
			}
		}
		if !categorized {
			categories["Core Commands"] = append(categories["Core Commands"], f)
		}
	}

	// Sort features within each category
	for cat := range categories {
		sort.Slice(categories[cat], func(i, j int) bool {
			return categories[cat][i].FilePath < categories[cat][j].FilePath
		})
	}

	// Build ordered result
	var result []featureGroup
	for _, cat := range categoryOrder {
		if len(categories[cat]) > 0 {
			result = append(result, featureGroup{
				name:     cat,
				features: categories[cat],
			})
		}
	}

	return result
}

func generateHTML(data DocData, outputFile string) error {
	tmpl, err := template.New("docs").Funcs(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"split": strings.Split,
	}).Parse(htmlTemplate)
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
            --color-bg: #0f172a;
            --color-surface: #1e293b;
            --color-surface-hover: #334155;
            --color-border: #334155;
            --color-text: #e2e8f0;
            --color-text-muted: #94a3b8;
            --color-keyword: #818cf8;
            --color-tag: #a78bfa;
            --color-accent: #38bdf8;
            --color-success: #22c55e;
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
        }
        .container {
            display: flex;
            min-height: 100vh;
        }
        /* Sidebar */
        .sidebar {
            width: 280px;
            background: var(--color-surface);
            border-right: 1px solid var(--color-border);
            padding: 1.5rem;
            position: fixed;
            height: 100vh;
            overflow-y: auto;
        }
        .sidebar h1 {
            font-size: 1.25rem;
            margin-bottom: 0.5rem;
        }
        .sidebar-meta {
            color: var(--color-text-muted);
            font-size: 0.75rem;
            margin-bottom: 1.5rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--color-border);
        }
        .sidebar-stats {
            display: flex;
            gap: 1rem;
            margin-bottom: 1.5rem;
        }
        .sidebar-stat {
            text-align: center;
        }
        .sidebar-stat-value {
            font-size: 1.5rem;
            font-weight: 700;
            color: var(--color-accent);
        }
        .sidebar-stat-label {
            font-size: 0.75rem;
            color: var(--color-text-muted);
        }
        .nav-group {
            margin-bottom: 1rem;
        }
        .nav-group-title {
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--color-text-muted);
            margin-bottom: 0.5rem;
        }
        .nav-link {
            display: block;
            padding: 0.375rem 0.5rem;
            color: var(--color-text);
            text-decoration: none;
            font-size: 0.875rem;
            border-radius: 0.25rem;
            margin-bottom: 0.125rem;
        }
        .nav-link:hover {
            background: var(--color-surface-hover);
        }
        .nav-link-count {
            color: var(--color-text-muted);
            font-size: 0.75rem;
            margin-left: 0.25rem;
        }
        /* Main content */
        .main {
            flex: 1;
            margin-left: 280px;
            padding: 2rem;
            max-width: calc(100% - 280px);
        }
        .feature {
            background: var(--color-surface);
            border-radius: 0.5rem;
            margin-bottom: 1.5rem;
            overflow: hidden;
        }
        .feature-header {
            padding: 1.25rem 1.5rem;
            border-bottom: 1px solid var(--color-border);
        }
        .feature-title {
            font-size: 1.25rem;
            font-weight: 600;
            margin-bottom: 0.25rem;
        }
        .feature-file {
            font-size: 0.75rem;
            color: var(--color-text-muted);
            font-family: 'SF Mono', Monaco, Consolas, monospace;
        }
        .feature-description {
            color: var(--color-text-muted);
            font-size: 0.875rem;
            margin-top: 0.75rem;
            white-space: pre-line;
        }
        .feature-tags {
            margin-top: 0.5rem;
        }
        .tag {
            display: inline-block;
            background: rgba(167, 139, 250, 0.15);
            color: var(--color-tag);
            padding: 0.125rem 0.5rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            margin-right: 0.25rem;
        }
        .feature-content {
            padding: 1rem 1.5rem;
        }
        .background {
            background: rgba(56, 189, 248, 0.1);
            border-left: 3px solid var(--color-accent);
            padding: 1rem;
            border-radius: 0.25rem;
            margin-bottom: 1rem;
        }
        .background-title {
            font-size: 0.875rem;
            font-weight: 600;
            color: var(--color-accent);
            margin-bottom: 0.5rem;
        }
        .scenario {
            border: 1px solid var(--color-border);
            border-radius: 0.375rem;
            margin-bottom: 0.75rem;
        }
        .scenario-header {
            padding: 0.75rem 1rem;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            background: rgba(0, 0, 0, 0.2);
        }
        .scenario-header:hover {
            background: rgba(0, 0, 0, 0.3);
        }
        .scenario-title {
            font-weight: 500;
        }
        .scenario-outline-badge {
            background: rgba(129, 140, 248, 0.2);
            color: var(--color-keyword);
            padding: 0.125rem 0.5rem;
            border-radius: 9999px;
            font-size: 0.75rem;
            margin-left: 0.5rem;
        }
        .scenario-tags {
            font-size: 0.75rem;
            color: var(--color-tag);
        }
        .scenario-content {
            padding: 1rem;
            display: none;
        }
        .scenario.expanded .scenario-content {
            display: block;
        }
        .step {
            padding: 0.5rem 0.75rem;
            font-family: 'SF Mono', Monaco, Consolas, monospace;
            font-size: 0.8125rem;
            border-radius: 0.25rem;
            margin-bottom: 0.25rem;
            background: rgba(255, 255, 255, 0.03);
        }
        .step-keyword {
            color: var(--color-keyword);
            font-weight: 600;
        }
        .step-extra {
            margin-top: 0.5rem;
            padding: 0.75rem;
            background: rgba(0, 0, 0, 0.3);
            border-radius: 0.25rem;
            font-size: 0.75rem;
        }
        .docstring {
            white-space: pre-wrap;
            color: var(--color-text-muted);
        }
        .data-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.75rem;
        }
        .data-table th,
        .data-table td {
            border: 1px solid var(--color-border);
            padding: 0.375rem 0.5rem;
            text-align: left;
        }
        .data-table th {
            background: rgba(0, 0, 0, 0.3);
            font-weight: 600;
        }
        .examples {
            margin-top: 1rem;
        }
        .examples-title {
            font-size: 0.875rem;
            font-weight: 600;
            color: var(--color-keyword);
            margin-bottom: 0.5rem;
        }
        .section-title {
            font-size: 1.5rem;
            font-weight: 600;
            margin-bottom: 1.5rem;
            padding-bottom: 0.75rem;
            border-bottom: 1px solid var(--color-border);
        }
        .expand-all {
            background: var(--color-surface);
            border: 1px solid var(--color-border);
            color: var(--color-text);
            padding: 0.5rem 1rem;
            border-radius: 0.375rem;
            cursor: pointer;
            font-size: 0.875rem;
            margin-bottom: 1.5rem;
        }
        .expand-all:hover {
            background: var(--color-surface-hover);
        }
        /* Search */
        .search-box {
            margin-bottom: 1rem;
        }
        .search-input {
            width: 100%;
            padding: 0.5rem 0.75rem;
            background: var(--color-bg);
            border: 1px solid var(--color-border);
            border-radius: 0.375rem;
            color: var(--color-text);
            font-size: 0.875rem;
        }
        .search-input:focus {
            outline: none;
            border-color: var(--color-accent);
        }
        .search-input::placeholder {
            color: var(--color-text-muted);
        }
        .hidden {
            display: none !important;
        }
        @media (max-width: 768px) {
            .sidebar {
                width: 100%;
                position: relative;
                height: auto;
            }
            .main {
                margin-left: 0;
                max-width: 100%;
            }
            .container {
                flex-direction: column;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <nav class="sidebar">
            <h1>{{.Title}}</h1>
            <p class="sidebar-meta">Generated: {{.GeneratedAt}}</p>

            <div class="sidebar-stats">
                <div class="sidebar-stat">
                    <div class="sidebar-stat-value">{{.TotalFeatures}}</div>
                    <div class="sidebar-stat-label">Features</div>
                </div>
                <div class="sidebar-stat">
                    <div class="sidebar-stat-value">{{.TotalScenarios}}</div>
                    <div class="sidebar-stat-label">Scenarios</div>
                </div>
            </div>

            <div class="search-box">
                <input type="text" class="search-input" placeholder="Search features..." onkeyup="filterFeatures(this.value)">
            </div>

            {{range .FeaturesByPhase}}
            <div class="nav-group">
                <div class="nav-group-title">{{.Name}}</div>
                {{range .Features}}
                <a href="#{{.FilePath}}" class="nav-link" data-feature="{{.Name}}">
                    {{.Name}}<span class="nav-link-count">({{.ScenarioCount}})</span>
                </a>
                {{end}}
            </div>
            {{end}}
        </nav>

        <main class="main">
            <button class="expand-all" onclick="toggleAllScenarios()">Expand/Collapse All Scenarios</button>

            {{range .FeaturesByPhase}}
            <h2 class="section-title">{{.Name}}</h2>

            {{range .Features}}
            <div class="feature" id="{{.FilePath}}" data-name="{{.Name}}">
                <div class="feature-header">
                    <h3 class="feature-title">{{.Name}}</h3>
                    <div class="feature-file">{{.FilePath}}</div>
                    {{if .Description}}
                    <p class="feature-description">{{.Description}}</p>
                    {{end}}
                    {{if .Tags}}
                    <div class="feature-tags">
                        {{range $tag := (split .Tags " ")}}
                        {{if $tag}}<span class="tag">{{$tag}}</span>{{end}}
                        {{end}}
                    </div>
                    {{end}}
                </div>
                <div class="feature-content">
                    {{if .Background}}
                    <div class="background">
                        <div class="background-title">Background</div>
                        {{range .Background.Steps}}
                        <div class="step">
                            <span class="step-keyword">{{.Keyword}}</span> {{.Text}}
                            {{if .HasExtra}}
                            <div class="step-extra">
                                {{if .DocString}}<pre class="docstring">{{.DocString}}</pre>{{end}}
                                {{if .DataTable}}
                                <table class="data-table">
                                    {{range $i, $row := .DataTable}}
                                    <tr>
                                        {{range $cell := $row}}
                                        {{if eq $i 0}}<th>{{$cell}}</th>{{else}}<td>{{$cell}}</td>{{end}}
                                        {{end}}
                                    </tr>
                                    {{end}}
                                </table>
                                {{end}}
                            </div>
                            {{end}}
                        </div>
                        {{end}}
                    </div>
                    {{end}}

                    {{range .Scenarios}}
                    <div class="scenario">
                        <div class="scenario-header" onclick="this.parentElement.classList.toggle('expanded')">
                            <div>
                                <span class="scenario-title">{{.Name}}</span>
                                {{if .IsOutline}}<span class="scenario-outline-badge">Outline</span>{{end}}
                                {{if .Tags}}<div class="scenario-tags">{{.Tags}}</div>{{end}}
                            </div>
                        </div>
                        <div class="scenario-content">
                            {{range .Steps}}
                            <div class="step">
                                <span class="step-keyword">{{.Keyword}}</span> {{.Text}}
                                {{if .HasExtra}}
                                <div class="step-extra">
                                    {{if .DocString}}<pre class="docstring">{{.DocString}}</pre>{{end}}
                                    {{if .DataTable}}
                                    <table class="data-table">
                                        {{range $i, $row := .DataTable}}
                                        <tr>
                                            {{range $cell := $row}}
                                            {{if eq $i 0}}<th>{{$cell}}</th>{{else}}<td>{{$cell}}</td>{{end}}
                                            {{end}}
                                        </tr>
                                        {{end}}
                                    </table>
                                    {{end}}
                                </div>
                                {{end}}
                            </div>
                            {{end}}

                            {{if .Examples}}
                            <div class="examples">
                                {{range .Examples}}
                                <div class="examples-title">Examples{{if .Name}}: {{.Name}}{{end}}</div>
                                <table class="data-table">
                                    <tr>
                                        {{range .Headers}}<th>{{.}}</th>{{end}}
                                    </tr>
                                    {{range .Rows}}
                                    <tr>
                                        {{range .}}<td>{{.}}</td>{{end}}
                                    </tr>
                                    {{end}}
                                </table>
                                {{end}}
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
            {{end}}
        </main>
    </div>

    <script>
        function toggleAllScenarios() {
            const scenarios = document.querySelectorAll('.scenario');
            const allExpanded = Array.from(scenarios).every(s => s.classList.contains('expanded'));
            scenarios.forEach(s => {
                if (allExpanded) {
                    s.classList.remove('expanded');
                } else {
                    s.classList.add('expanded');
                }
            });
        }

        function filterFeatures(query) {
            query = query.toLowerCase();
            const features = document.querySelectorAll('.feature');
            const navLinks = document.querySelectorAll('.nav-link');

            features.forEach(f => {
                const name = f.dataset.name.toLowerCase();
                const content = f.textContent.toLowerCase();
                if (query === '' || name.includes(query) || content.includes(query)) {
                    f.classList.remove('hidden');
                } else {
                    f.classList.add('hidden');
                }
            });

            navLinks.forEach(link => {
                const name = link.dataset.feature.toLowerCase();
                if (query === '' || name.includes(query)) {
                    link.classList.remove('hidden');
                } else {
                    link.classList.add('hidden');
                }
            });
        }

        // Smooth scroll to anchor
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', function(e) {
                e.preventDefault();
                const target = document.querySelector(this.getAttribute('href'));
                if (target) {
                    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                }
            });
        });
    </script>
</body>
</html>
`


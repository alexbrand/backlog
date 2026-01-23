package spec

import (
	"io"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	"github.com/alexbrand/backlog/spec/steps"
)

func TestFeatures(t *testing.T) {
	// Determine output format and destination
	format := "pretty"
	var output io.Writer = colors.Colored(os.Stdout)

	// Support cucumber JSON output for HTML report generation
	if jsonFile := os.Getenv("GODOG_JSON_OUTPUT"); jsonFile != "" {
		format = "cucumber"
		f, err := os.Create(jsonFile)
		if err != nil {
			t.Fatalf("failed to create JSON output file: %v", err)
		}
		defer f.Close()
		output = f
	}

	// Allow format override via environment variable
	if envFormat := os.Getenv("GODOG_FORMAT"); envFormat != "" {
		format = envFormat
	}

	// Determine tags - allow override via environment variable
	tags := "~@remote" // Exclude remote backend tests by default
	if envTags := os.Getenv("GODOG_TAGS"); envTags != "" {
		tags = envTags
	}

	opts := godog.Options{
		Output:      output,
		Format:      format,
		Paths:       []string{"features"},
		Randomize:   0,
		Concurrency: 1,
		Tags:        tags,
	}

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	// Register common step definitions
	steps.InitializeCommonSteps(ctx)
}

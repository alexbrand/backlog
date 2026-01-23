package spec

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	"github.com/alexbrand/backlog/spec/steps"
)

func TestFeatures(t *testing.T) {
	opts := godog.Options{
		Output:      colors.Colored(os.Stdout),
		Format:      "pretty",
		Paths:       []string{"features"},
		Randomize:   0,
		Concurrency: 1,
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

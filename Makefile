# Makefile for backlog CLI

.PHONY: spec spec-local spec-github spec-linear spec-all spec-coverage spec-coverage-html spec-report spec-report-html

# Run Gherkin specs (excludes @remote tests by default)
spec:
	cd spec && go test -run TestFeatures -v .

# Run only local backend specs (no remote backend tests)
spec-local:
	cd spec && go test -run TestFeatures -v .

# Run GitHub backend specs (requires mock server)
spec-github:
	cd spec && GODOG_TAGS="@github" go test -run TestFeatures -v .

# Run Linear backend specs (requires mock server)
spec-linear:
	cd spec && GODOG_TAGS="@linear" go test -run TestFeatures -v .

# Run all specs including remote backend tests
spec-all:
	cd spec && GODOG_TAGS="" go test -run TestFeatures -v .

# Run specs with coverage reporting
spec-coverage:
	cd spec && GODOG_TAGS="" go test -run TestFeatures -v -cover -coverprofile=coverage.out -coverpkg=../... .
	@echo ""
	@echo "Coverage summary:"
	@cd spec && go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
spec-coverage-html: spec-coverage
	cd spec && go tool cover -html=coverage.out -o coverage.html
	@echo "HTML coverage report generated: spec/coverage.html"

# Generate Cucumber JSON report from specs (allows failures since tests document expected behavior)
spec-report:
	-cd spec && GODOG_TAGS="" GODOG_JSON_OUTPUT=cucumber.json go test -run TestFeatures -v .
	@echo "Cucumber JSON report generated: spec/cucumber.json"

# Generate HTML spec report for documentation
spec-report-html: spec-report
	cd spec && go run ./cmd/genreport -input cucumber.json -output report.html
	@echo "HTML spec report generated: spec/report.html"

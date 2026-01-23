# Makefile for backlog CLI

.PHONY: spec spec-local spec-github spec-linear spec-all spec-coverage spec-coverage-html

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

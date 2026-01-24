# Makefile for backlog CLI

# Build variables
BINARY_NAME := backlog
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X github.com/alexbrand/backlog/internal/cli.Version=$(VERSION) -X github.com/alexbrand/backlog/internal/cli.GitCommit=$(COMMIT) -X github.com/alexbrand/backlog/internal/cli.BuildDate=$(BUILD_TIME)"

# Output directory
DIST_DIR := dist

.PHONY: build build-all build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-linux-arm64 build-windows-amd64
.PHONY: clean test lint install
.PHONY: spec spec-local spec-github spec-linear spec-all spec-coverage spec-coverage-html spec-report spec-report-html spec-docs

# Build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/backlog

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/backlog

# Build for all platforms
build-all: build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-linux-arm64 build-windows-amd64
	@echo "All builds complete. Binaries in $(DIST_DIR)/"

# macOS ARM64 (Apple Silicon)
build-darwin-arm64:
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/backlog

# macOS AMD64 (Intel)
build-darwin-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/backlog

# Linux AMD64
build-linux-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/backlog

# Linux ARM64
build-linux-arm64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/backlog

# Windows AMD64
build-windows-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/backlog

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR)
	rm -f spec/coverage.out spec/coverage.html spec/cucumber.json spec/report.html

# Run tests
test:
	go test -v ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

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

# Generate living documentation from feature files (no test execution required)
spec-docs:
	cd spec && go run ./cmd/gendocs -features features -output docs.html
	@echo "Living documentation generated: spec/docs.html"

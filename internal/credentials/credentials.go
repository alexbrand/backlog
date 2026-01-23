// Package credentials provides secure credential loading and management.
// Credentials are stored in ~/.config/backlog/credentials.yaml with 0600 permissions.
package credentials

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Credentials represents the top-level credentials structure.
type Credentials struct {
	GitHub *GitHubCredentials `yaml:"github,omitempty"`
	Linear *LinearCredentials `yaml:"linear,omitempty"`
}

// GitHubCredentials holds GitHub-specific credentials.
type GitHubCredentials struct {
	Token string `yaml:"token"`
}

// LinearCredentials holds Linear-specific credentials.
type LinearCredentials struct {
	APIKey string `yaml:"api_key"`
}

var (
	creds     *Credentials
	credsFile string
)

// configDir returns the configuration directory path.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "backlog"), nil
}

// DefaultCredentialsPath returns the default credentials file path.
func DefaultCredentialsPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.yaml"), nil
}

// Init initializes the credentials system by loading credentials from file.
// If the credentials file doesn't exist, an empty credentials struct is used.
// This is not an error - credentials may come from environment variables.
func Init() error {
	credPath, err := DefaultCredentialsPath()
	if err != nil {
		return err
	}
	credsFile = credPath

	// Check if file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		// File doesn't exist - use empty credentials
		creds = &Credentials{}
		return nil
	}

	// Read and parse credentials file
	data, err := os.ReadFile(credPath)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	creds = &Credentials{}
	if err := yaml.Unmarshal(data, creds); err != nil {
		return fmt.Errorf("failed to parse credentials file: %w", err)
	}

	return nil
}

// Get returns the current credentials.
// Returns nil if Init has not been called.
func Get() *Credentials {
	return creds
}

// GetGitHubToken returns the GitHub token using the following priority:
// 1. GITHUB_TOKEN environment variable
// 2. credentials.yaml github.token
// Returns an error if no token is found.
func GetGitHubToken() (string, error) {
	// Check environment variable first
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}

	// Check credentials file
	if creds != nil && creds.GitHub != nil && creds.GitHub.Token != "" {
		return creds.GitHub.Token, nil
	}

	return "", errors.New("GitHub token not found: set GITHUB_TOKEN environment variable or add token to ~/.config/backlog/credentials.yaml")
}

// GetLinearAPIKey returns the Linear API key using the following priority:
// 1. LINEAR_API_KEY environment variable
// 2. credentials.yaml linear.api_key
// Returns an error if no API key is found.
func GetLinearAPIKey() (string, error) {
	// Check environment variable first
	if key := os.Getenv("LINEAR_API_KEY"); key != "" {
		return key, nil
	}

	// Check credentials file
	if creds != nil && creds.Linear != nil && creds.Linear.APIKey != "" {
		return creds.Linear.APIKey, nil
	}

	return "", errors.New("Linear API key not found: set LINEAR_API_KEY environment variable or add api_key to ~/.config/backlog/credentials.yaml")
}

// SaveGitHubToken saves a GitHub token to the credentials file.
// Creates the file with 0600 permissions if it doesn't exist.
func SaveGitHubToken(token string) error {
	return saveCredential(func(c *Credentials) {
		if c.GitHub == nil {
			c.GitHub = &GitHubCredentials{}
		}
		c.GitHub.Token = token
	})
}

// SaveLinearAPIKey saves a Linear API key to the credentials file.
// Creates the file with 0600 permissions if it doesn't exist.
func SaveLinearAPIKey(apiKey string) error {
	return saveCredential(func(c *Credentials) {
		if c.Linear == nil {
			c.Linear = &LinearCredentials{}
		}
		c.Linear.APIKey = apiKey
	})
}

// saveCredential saves credentials after applying the given update function.
func saveCredential(updateFn func(*Credentials)) error {
	credPath, err := DefaultCredentialsPath()
	if err != nil {
		return err
	}

	// Load existing credentials or create new
	currentCreds := &Credentials{}
	if data, err := os.ReadFile(credPath); err == nil {
		yaml.Unmarshal(data, currentCreds)
	}

	// Apply update
	updateFn(currentCreds)

	// Marshal to YAML
	data, err := yaml.Marshal(currentCreds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(credPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Write with secure permissions (owner read/write only)
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	// Update in-memory credentials
	creds = currentCreds

	return nil
}

// CredentialsFilePath returns the path to the credentials file being used.
func CredentialsFilePath() string {
	return credsFile
}

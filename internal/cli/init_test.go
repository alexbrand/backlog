package cli

import "testing"

func TestParseGitHubRepoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH format with .git",
			url:      "git@github.com:owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "SSH format without .git",
			url:      "git@github.com:owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "HTTPS format with .git",
			url:      "https://github.com/owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "HTTPS format without .git",
			url:      "https://github.com/owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "SSH format with hyphenated names",
			url:      "git@github.com:my-org/my-repo.git",
			expected: "my-org/my-repo",
		},
		{
			name:     "HTTPS format with hyphenated names",
			url:      "https://github.com/my-org/my-repo.git",
			expected: "my-org/my-repo",
		},
		{
			name:     "Non-GitHub SSH URL",
			url:      "git@gitlab.com:owner/repo.git",
			expected: "",
		},
		{
			name:     "Non-GitHub HTTPS URL",
			url:      "https://gitlab.com/owner/repo.git",
			expected: "",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitHubRepoFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("parseGitHubRepoFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

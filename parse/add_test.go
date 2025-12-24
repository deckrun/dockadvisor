package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidADDFlag(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		expected bool
	}{
		// Valid flags
		{
			name:     "--keep-git-dir flag",
			flag:     "--keep-git-dir",
			expected: true,
		},
		{
			name:     "--checksum flag",
			flag:     "--checksum",
			expected: true,
		},
		{
			name:     "--chown flag",
			flag:     "--chown",
			expected: true,
		},
		{
			name:     "--chmod flag",
			flag:     "--chmod",
			expected: true,
		},
		{
			name:     "--link flag",
			flag:     "--link",
			expected: true,
		},
		{
			name:     "--exclude flag",
			flag:     "--exclude",
			expected: true,
		},
		// Invalid flags
		{
			name:     "invalid flag",
			flag:     "--invalid",
			expected: false,
		},
		{
			name:     "from flag (valid for COPY, not ADD)",
			flag:     "--from",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidADDFlag(tt.flag)
			require.Equal(t, tt.expected, result, "isValidADDFlag(%q) returned unexpected result", tt.flag)
		})
	}
}

func TestParseADD(t *testing.T) {
	tests := []struct {
		name          string
		dockerfile    string
		expectedRules []string // Rule codes expected to be present
	}{
		// Valid ADD instructions
		{
			name:          "simple add",
			dockerfile:    `FROM alpine\nADD file.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "multiple source files",
			dockerfile:    `FROM alpine\nADD file1.txt file2.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "wildcard pattern",
			dockerfile:    `FROM alpine\nADD *.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "URL source",
			dockerfile:    `FROM alpine\nADD https://example.com/file.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "tar archive",
			dockerfile:    `FROM alpine\nADD archive.tar.gz /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "with --chown flag",
			dockerfile:    `FROM alpine\nADD --chown=user:group file.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "with --chmod flag",
			dockerfile:    `FROM alpine\nADD --chmod=755 file.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "with --link flag",
			dockerfile:    `FROM alpine\nADD --link file.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "with --checksum flag",
			dockerfile:    `FROM alpine\nADD --checksum=sha256:abc123 file.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "with --keep-git-dir flag",
			dockerfile:    `FROM alpine\nADD --keep-git-dir=true git@github.com:user/repo.git /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "with multiple flags",
			dockerfile:    `FROM alpine\nADD --chown=user:group --chmod=755 file.txt /dest/`,
			expectedRules: []string{},
		},
		{
			name:          "directory add",
			dockerfile:    `FROM alpine\nADD src/ /dest/`,
			expectedRules: []string{},
		},
		// Invalid ADD instructions
		{
			name: "no arguments",
			dockerfile: `FROM alpine
ADD`,
			expectedRules: []string{"InvalidInstruction"},
		},
		{
			name: "only source, no destination",
			dockerfile: `FROM alpine
ADD file.txt`,
			expectedRules: []string{"AddMissingArguments"},
		},
		{
			name: "only flag, no files",
			dockerfile: `FROM alpine
ADD --chown=user:group`,
			expectedRules: []string{"InvalidInstruction"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfile)
			require.NoError(t, err)

			// Check that expected rules are present
			actualRuleCodes := make(map[string]bool)
			for _, rule := range result.Rules {
				actualRuleCodes[rule.Code] = true
			}

			for _, expectedCode := range tt.expectedRules {
				require.True(t, actualRuleCodes[expectedCode], "Expected rule code %q not found in results. Got rules: %v", expectedCode, result.Rules)
			}

			// Check that we don't have extra unexpected rules (excluding other instruction rules)
			if len(tt.expectedRules) == 0 {
				// Only check for ADD-related rules
				for _, rule := range result.Rules {
					if strings.HasPrefix(rule.Code, "Add") {
						t.Errorf("Expected no ADD rules but got: %v", rule)
					}
				}
			}
		})
	}
}

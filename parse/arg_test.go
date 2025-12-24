package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidARGName(t *testing.T) {
	tests := []struct {
		name     string
		argName  string
		expected bool
	}{
		// Valid names
		{
			name:     "simple lowercase",
			argName:  "user",
			expected: true,
		},
		{
			name:     "simple uppercase",
			argName:  "USER",
			expected: true,
		},
		{
			name:     "with underscore",
			argName:  "user_name",
			expected: true,
		},
		{
			name:     "with numbers",
			argName:  "user1",
			expected: true,
		},
		{
			name:     "starts with underscore",
			argName:  "_user",
			expected: true,
		},
		{
			name:     "mixed case with numbers",
			argName:  "CONT_IMG_VER",
			expected: true,
		},
		// Invalid names
		{
			name:     "empty string",
			argName:  "",
			expected: false,
		},
		{
			name:     "starts with number",
			argName:  "1user",
			expected: false,
		},
		{
			name:     "contains dash",
			argName:  "user-name",
			expected: false,
		},
		{
			name:     "contains dot",
			argName:  "user.name",
			expected: false,
		},
		{
			name:     "contains special characters",
			argName:  "user@host",
			expected: false,
		},
		{
			name:     "contains space",
			argName:  "user name",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidARGName(tt.argName)
			require.Equal(t, tt.expected, result, "isValidARGName(%q) returned unexpected result", tt.argName)
		})
	}
}

func TestCheckARGFormat(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		// Valid formats
		{
			name:     "simple name",
			config:   "user",
			expected: true,
		},
		{
			name:     "name with default value",
			config:   "user=someuser",
			expected: true,
		},
		{
			name:     "name with empty default",
			config:   "user=",
			expected: true,
		},
		{
			name:     "multiple args",
			config:   "user1 buildno",
			expected: true,
		},
		{
			name:     "multiple args with defaults",
			config:   "user1=someuser buildno=1",
			expected: true,
		},
		{
			name:     "mixed args with and without defaults",
			config:   "user1 buildno=1",
			expected: true,
		},
		{
			name:     "uppercase name",
			config:   "CONT_IMG_VER",
			expected: true,
		},
		{
			name:     "uppercase with default",
			config:   "CONT_IMG_VER=v1.0.0",
			expected: true,
		},
		// Invalid formats
		{
			name:     "empty string",
			config:   "",
			expected: false,
		},
		{
			name:     "name starting with number",
			config:   "1user",
			expected: false,
		},
		{
			name:     "name with dash",
			config:   "user-name",
			expected: false,
		},
		{
			name:     "name with dot",
			config:   "user.name",
			expected: false,
		},
		{
			name:     "only equals sign",
			config:   "=",
			expected: false,
		},
		{
			name:     "starts with equals",
			config:   "=value",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkARGFormat(tt.config)
			require.Equal(t, tt.expected, result, "checkARGFormat(%q) returned unexpected result", tt.config)
		})
	}
}

func TestParseARG(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid ARG instructions
		{
			name:              "simple name",
			dockerfileContent: `ARG user`,
			expectedRules:     []string{},
		},
		{
			name:              "name with default value",
			dockerfileContent: `ARG user=someuser`,
			expectedRules:     []string{},
		},
		{
			name:              "name with empty default",
			dockerfileContent: `ARG user=`,
			expectedRules:     []string{},
		},
		{
			name:              "multiple args",
			dockerfileContent: `ARG user1 buildno`,
			expectedRules:     []string{"LegacyKeyValueFormat"},
		},
		{
			name:              "multiple args with defaults",
			dockerfileContent: `ARG user1=someuser buildno=1`,
			expectedRules:     []string{},
		},
		{
			name:              "mixed args",
			dockerfileContent: `ARG user1 buildno=1`,
			expectedRules:     []string{"LegacyKeyValueFormat"},
		},
		{
			name:              "uppercase name",
			dockerfileContent: `ARG CONT_IMG_VER`,
			expectedRules:     []string{},
		},
		{
			name:              "uppercase with default",
			dockerfileContent: `ARG CONT_IMG_VER=v1.0.0`,
			expectedRules:     []string{},
		},
		// Invalid ARG instructions
		{
			name:              "no arguments",
			dockerfileContent: `ARG`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "name starting with number",
			dockerfileContent: `ARG 1user`,
			expectedRules:     []string{"ArgInvalidFormat"},
		},
		{
			name:              "name with dash",
			dockerfileContent: `ARG user-name`,
			expectedRules:     []string{"ArgInvalidFormat"},
		},
		{
			name:              "name with dot",
			dockerfileContent: `ARG user.name`,
			expectedRules:     []string{"ArgInvalidFormat"},
		},
		{
			name:              "only equals sign",
			dockerfileContent: `ARG =`,
			expectedRules:     []string{"ArgInvalidFormat"},
		},
		{
			name:              "starts with equals",
			dockerfileContent: `ARG =value`,
			expectedRules:     []string{"ArgInvalidFormat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error for valid Dockerfile")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Check that we don't have extra unexpected rules
			if len(tt.expectedRules) == 0 {
				require.Empty(t, result.Rules, "Expected no rules but got: %v", result.Rules)
			} else {
				require.Equal(t, len(tt.expectedRules), len(result.Rules), "Number of rules doesn't match expected. Got: %v", result.Rules)

				// Check that expected rules are present
				actualRuleCodes := make(map[string]bool)
				for _, rule := range result.Rules {
					actualRuleCodes[rule.Code] = true
				}

				for _, expectedCode := range tt.expectedRules {
					require.True(t, actualRuleCodes[expectedCode], "Expected rule code %q not found in results. Got rules: %v", expectedCode, result.Rules)
				}
			}
		})
	}
}

func TestARGLegacyFormat(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
	}{
		// Invalid cases - SHOULD flag violations
		{
			name: "legacy format with space separator",
			dockerfileContent: `FROM alpine
ARG foo bar`,
			expectViolation: true,
		},
		{
			name: "legacy format with multiple spaces",
			dockerfileContent: `FROM alpine
ARG VERSION   1.0.0`,
			expectViolation: true,
		},
		{
			name: "legacy format with complex value",
			dockerfileContent: `FROM alpine
ARG BUILD_DATE 2024-01-01`,
			expectViolation: true,
		},

		// Valid cases - should NOT flag violations
		{
			name: "correct format with equals",
			dockerfileContent: `FROM alpine
ARG foo=bar`,
			expectViolation: false,
		},
		{
			name: "ARG without default value",
			dockerfileContent: `FROM alpine
ARG TAG`,
			expectViolation: false,
		},
		{
			name: "ARG with default value using equals",
			dockerfileContent: `FROM alpine
ARG VERSION=1.0.0`,
			expectViolation: false,
		},
		{
			name: "ARG with empty default",
			dockerfileContent: `FROM alpine
ARG ENV_VAR=`,
			expectViolation: false,
		},
		{
			name: "multiple ARGs with equals",
			dockerfileContent: `FROM alpine
ARG VERSION=1.0.0
ARG BUILD_DATE=2024-01-01`,
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for LegacyKeyValueFormat rules
			legacyRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "LegacyKeyValueFormat" {
					legacyRules = append(legacyRules, rule)
				}
			}

			if tt.expectViolation {
				require.NotEmpty(t, legacyRules,
					"Expected LegacyKeyValueFormat violation but got none")

				// Verify rule structure
				for _, rule := range legacyRules {
					require.Equal(t, "LegacyKeyValueFormat", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "Legacy key/value format")
					require.Contains(t, rule.Description, "ARG key=value")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/legacy-key-value-format/",
						rule.Url)
				}
			} else {
				require.Empty(t, legacyRules,
					"Expected no LegacyKeyValueFormat violations but got: %v", legacyRules)
			}
		})
	}
}

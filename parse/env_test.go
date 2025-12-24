package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckENVLegacySyntax(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		// Legacy syntax (space-separated)
		{
			name:     "legacy syntax simple",
			config:   "MY_VAR my-value",
			expected: true,
		},
		{
			name:     "legacy syntax with quoted value",
			config:   `MY_NAME "John Doe"`,
			expected: true,
		},
		// Modern syntax (key=value)
		{
			name:     "modern syntax simple",
			config:   "MY_VAR=my-value",
			expected: false,
		},
		{
			name:     "modern syntax with quoted value",
			config:   `MY_NAME="John Doe"`,
			expected: false,
		},
		{
			name:     "modern syntax multiple vars",
			config:   "VAR1=value1 VAR2=value2",
			expected: false,
		},
		// Edge cases
		{
			name:     "empty string",
			config:   "",
			expected: false,
		},
		{
			name:     "just key no space or equals",
			config:   "MY_VAR",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkENVLegacySyntax(tt.config)
			require.Equal(t, tt.expected, result, "checkENVLegacySyntax(%q) returned unexpected result", tt.config)
		})
	}
}

func TestCheckENVFormat(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		// Valid formats
		{
			name:     "simple key=value",
			config:   "MY_VAR=value",
			expected: true,
		},
		{
			name:     "quoted value",
			config:   `MY_NAME="John Doe"`,
			expected: true,
		},
		{
			name:     "multiple vars",
			config:   "VAR1=value1 VAR2=value2 VAR3=value3",
			expected: true,
		},
		{
			name:     "empty value",
			config:   "MY_VAR=",
			expected: true,
		},
		{
			name:     "value with spaces",
			config:   `MY_DOG=Rex\ The\ Dog`,
			expected: true,
		},
		// Invalid formats
		{
			name:     "empty string",
			config:   "",
			expected: false,
		},
		{
			name:     "no equals sign",
			config:   "MY_VAR value",
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
			result := checkENVFormat(tt.config)
			require.Equal(t, tt.expected, result, "checkENVFormat(%q) returned unexpected result", tt.config)
		})
	}
}

func TestParseENV(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid ENV instructions (modern syntax)
		{
			name:              "simple key=value",
			dockerfileContent: `ENV MY_VAR=value`,
			expectedRules:     []string{},
		},
		{
			name:              "quoted value",
			dockerfileContent: `ENV MY_NAME="John Doe"`,
			expectedRules:     []string{},
		},
		{
			name:              "multiple vars on one line",
			dockerfileContent: `ENV VAR1=value1 VAR2=value2 VAR3=value3`,
			expectedRules:     []string{},
		},
		{
			name:              "empty value",
			dockerfileContent: `ENV MY_VAR=`,
			expectedRules:     []string{},
		},
		{
			name:              "value with escaped spaces",
			dockerfileContent: `ENV MY_DOG=Rex\ The\ Dog`,
			expectedRules:     []string{},
		},
		// Legacy syntax (discouraged)
		{
			name:              "legacy syntax simple",
			dockerfileContent: `ENV MY_VAR my-value`,
			expectedRules:     []string{"LegacyKeyValueFormat"},
		},
		{
			name:              "legacy syntax with quoted value",
			dockerfileContent: `ENV MY_NAME "John Doe"`,
			expectedRules:     []string{"LegacyKeyValueFormat"},
		},
		{
			name: "legacy syntax multi-line with continuation",
			dockerfileContent: `ENV DEPS \
    curl \
    git \
    make`,
			expectedRules: []string{"LegacyKeyValueFormat"},
		},
		{
			name: "modern syntax multi-line with continuation",
			dockerfileContent: `ENV DEPS="\
    curl \
    git \
    make"`,
			expectedRules: []string{},
		},
		// Invalid ENV instructions
		{
			name:              "no arguments",
			dockerfileContent: `ENV`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "only equals sign",
			dockerfileContent: `ENV =`,
			expectedRules:     []string{"EnvInvalidFormat"},
		},
		{
			name:              "starts with equals",
			dockerfileContent: `ENV =value`,
			expectedRules:     []string{"EnvInvalidFormat"},
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

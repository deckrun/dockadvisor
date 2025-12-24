package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckLABELFormat(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		// Valid formats
		{
			name:     "simple key=value",
			config:   "version=1.0",
			expected: true,
		},
		{
			name:     "key with dots",
			config:   "com.example.vendor=ACME",
			expected: true,
		},
		{
			name:     "key with dashes",
			config:   "com.example.label-with-value=foo",
			expected: true,
		},
		{
			name:     "multiple labels",
			config:   "multi.label1=value1 multi.label2=value2",
			expected: true,
		},
		{
			name:     "quoted value with spaces",
			config:   `description="This is a description"`,
			expected: true,
		},
		{
			name:     "value with variable",
			config:   `example=foo-$ENV_VAR`,
			expected: true,
		},
		{
			name:     "empty value",
			config:   "key=",
			expected: true,
		},
		{
			name:     "multiple key=value pairs",
			config:   "key1=value1 key2=value2 key3=value3",
			expected: true,
		},
		// Invalid formats
		{
			name:     "empty string",
			config:   "",
			expected: false,
		},
		{
			name:     "whitespace only",
			config:   "   ",
			expected: false,
		},
		{
			name:     "no equals sign",
			config:   "version 1.0",
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
		{
			name:     "just a key",
			config:   "version",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkLABELFormat(tt.config)
			require.Equal(t, tt.expected, result, "checkLABELFormat(%q) returned unexpected result", tt.config)
		})
	}
}

func TestParseLABEL(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid LABEL instructions
		{
			name:              "simple key=value",
			dockerfileContent: `LABEL version=1.0`,
			expectedRules:     []string{},
		},
		{
			name:              "key with dots",
			dockerfileContent: `LABEL com.example.vendor=ACME`,
			expectedRules:     []string{},
		},
		{
			name:              "key with dashes",
			dockerfileContent: `LABEL com.example.label-with-value=foo`,
			expectedRules:     []string{},
		},
		{
			name:              "multiple labels on one line",
			dockerfileContent: `LABEL multi.label1=value1 multi.label2=value2 other=value3`,
			expectedRules:     []string{},
		},
		{
			name:              "quoted value with spaces",
			dockerfileContent: `LABEL description="This is a description"`,
			expectedRules:     []string{},
		},
		{
			name:              "value with variable",
			dockerfileContent: `LABEL example=foo-$ENV_VAR`,
			expectedRules:     []string{},
		},
		{
			name:              "empty value",
			dockerfileContent: `LABEL key=`,
			expectedRules:     []string{},
		},
		{
			name:              "org.opencontainers format",
			dockerfileContent: `LABEL org.opencontainers.image.authors=user@example.com`,
			expectedRules:     []string{},
		},
		// Invalid LABEL instructions
		{
			name:              "no arguments",
			dockerfileContent: `LABEL`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "no equals sign",
			dockerfileContent: `LABEL version 1.0`,
			expectedRules:     []string{"LabelInvalidFormat"},
		},
		{
			name:              "only equals sign",
			dockerfileContent: `LABEL =`,
			expectedRules:     []string{"LabelInvalidFormat"},
		},
		{
			name:              "starts with equals",
			dockerfileContent: `LABEL =value`,
			expectedRules:     []string{"LabelInvalidFormat"},
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

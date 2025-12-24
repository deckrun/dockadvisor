package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckENTRYPOINTExecFormJSON(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		// Valid exec forms
		{
			name:     "exec form with executable",
			command:  `["/usr/sbin/apache2ctl", "-D", "FOREGROUND"]`,
			expected: true,
		},
		{
			name:     "exec form with single command",
			command:  `["nginx"]`,
			expected: true,
		},
		{
			name:     "exec form with multiple args",
			command:  `["top", "-b"]`,
			expected: true,
		},
		// Invalid exec forms
		{
			name:     "empty array (invalid for ENTRYPOINT)",
			command:  `[]`,
			expected: false,
		},
		{
			name:     "single quotes instead of double",
			command:  `['nginx', '-g', 'daemon off;']`,
			expected: false,
		},
		{
			name:     "missing closing bracket",
			command:  `["nginx", "-g"`,
			expected: false,
		},
		{
			name:     "not valid JSON",
			command:  `[nginx, -g]`,
			expected: false,
		},
		{
			name:     "mixed quotes",
			command:  `["nginx", 'daemon off;']`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkENTRYPOINTExecFormJSON(tt.command)
			require.Equal(t, tt.expected, result, "checkENTRYPOINTExecFormJSON(%q) returned unexpected result", tt.command)
		})
	}
}

func TestParseENTRYPOINT(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid ENTRYPOINT instructions
		{
			name:              "simple shell form",
			dockerfileContent: `ENTRYPOINT echo hello`,
			expectedRules:     []string{"JSONArgsRecommended"},
		},
		{
			name:              "shell form with script",
			dockerfileContent: `ENTRYPOINT nginx -g daemon off;`,
			expectedRules:     []string{"JSONArgsRecommended"},
		},
		{
			name:              "valid exec form",
			dockerfileContent: `ENTRYPOINT ["nginx", "-g", "daemon off;"]`,
			expectedRules:     []string{},
		},
		{
			name:              "exec form with executable path",
			dockerfileContent: `ENTRYPOINT ["/usr/sbin/apache2ctl", "-D", "FOREGROUND"]`,
			expectedRules:     []string{},
		},
		{
			name:              "exec form with top command",
			dockerfileContent: `ENTRYPOINT ["top", "-b"]`,
			expectedRules:     []string{},
		},
		// Invalid ENTRYPOINT instructions
		{
			name:              "invalid exec form with single quotes",
			dockerfileContent: `ENTRYPOINT ['echo', 'hello']`,
			expectedRules:     []string{"EntrypointInvalidExecForm"},
		},
		{
			name:              "invalid exec form - not valid JSON",
			dockerfileContent: `ENTRYPOINT [echo, hello]`,
			expectedRules:     []string{"EntrypointInvalidExecForm"},
		},
		{
			name:              "invalid exec form - empty array",
			dockerfileContent: `ENTRYPOINT []`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "invalid exec form - missing closing bracket",
			dockerfileContent: `ENTRYPOINT ["nginx", "-g"`,
			expectedRules:     []string{"EntrypointInvalidExecForm"},
		},
		{
			name:              "no arguments",
			dockerfileContent: `ENTRYPOINT`,
			expectedRules:     []string{"InvalidInstruction"},
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

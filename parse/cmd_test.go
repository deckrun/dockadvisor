package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckCMDExecFormJSON(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		// Valid exec forms
		{
			name:     "simple exec form",
			command:  `["echo", "hello"]`,
			expected: true,
		},
		{
			name:     "exec form with single command",
			command:  `["/bin/bash"]`,
			expected: true,
		},
		{
			name:     "exec form with multiple args",
			command:  `["nginx", "-g", "daemon off;"]`,
			expected: true,
		},
		{
			name:     "exec form as default params to ENTRYPOINT",
			command:  `["param1", "param2"]`,
			expected: true,
		},
		{
			name:     "empty array (valid for CMD as default params)",
			command:  `[]`,
			expected: true,
		},
		// Invalid exec forms
		{
			name:     "single quotes instead of double",
			command:  `['echo', 'hello']`,
			expected: false,
		},
		{
			name:     "missing closing bracket",
			command:  `["echo", "hello"`,
			expected: false,
		},
		{
			name:     "not valid JSON",
			command:  `[echo, hello]`,
			expected: false,
		},
		{
			name:     "mixed quotes",
			command:  `["echo", 'hello']`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkCMDExecFormJSON(tt.command)
			require.Equal(t, tt.expected, result, "checkCMDExecFormJSON(%q) returned unexpected result", tt.command)
		})
	}
}

func TestParseCMD(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid CMD instructions (exec form - recommended)
		{
			name:              "valid exec form",
			dockerfileContent: `CMD ["echo", "hello"]`,
			expectedRules:     []string{},
		},
		{
			name:              "exec form with executable and params",
			dockerfileContent: `CMD ["nginx", "-g", "daemon off;"]`,
			expectedRules:     []string{},
		},
		{
			name:              "exec form as default params",
			dockerfileContent: `CMD ["param1", "param2"]`,
			expectedRules:     []string{},
		},
		{
			name:              "empty array as default params",
			dockerfileContent: `CMD []`,
			expectedRules:     []string{},
		},
		// Shell form (valid but not recommended)
		{
			name:              "simple shell form",
			dockerfileContent: `CMD echo hello`,
			expectedRules:     []string{"JSONArgsRecommended"},
		},
		{
			name:              "shell form with multiple commands",
			dockerfileContent: `CMD nginx -g daemon off;`,
			expectedRules:     []string{"JSONArgsRecommended"},
		},
		// Invalid CMD instructions
		{
			name:              "invalid exec form with single quotes",
			dockerfileContent: `CMD ['echo', 'hello']`,
			expectedRules:     []string{"CmdInvalidExecForm"},
		},
		{
			name:              "invalid exec form - not valid JSON",
			dockerfileContent: `CMD [echo, hello]`,
			expectedRules:     []string{"CmdInvalidExecForm"},
		},
		{
			name:              "invalid exec form - missing closing bracket",
			dockerfileContent: `CMD ["echo", "hello"`,
			expectedRules:     []string{"CmdInvalidExecForm"},
		},
		{
			name:              "no arguments",
			dockerfileContent: `CMD`,
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

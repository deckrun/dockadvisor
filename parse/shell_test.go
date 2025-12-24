package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSHELL(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid SHELL instructions
		{
			name:              "default Linux shell",
			dockerfileContent: `SHELL ["/bin/sh", "-c"]`,
			expectedRules:     []string{},
		},
		{
			name:              "default Windows shell",
			dockerfileContent: `SHELL ["cmd", "/S", "/C"]`,
			expectedRules:     []string{},
		},
		{
			name:              "PowerShell configuration",
			dockerfileContent: `SHELL ["powershell", "-command"]`,
			expectedRules:     []string{},
		},
		{
			name:              "single executable",
			dockerfileContent: `SHELL ["sh"]`,
			expectedRules:     []string{},
		},
		// Invalid SHELL instructions
		{
			name:              "invalid JSON - empty array",
			dockerfileContent: "SHELL []",
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "invalid JSON - single quotes",
			dockerfileContent: `SHELL ['/bin/sh', '-c']`,
			expectedRules:     []string{"ShellInvalidJsonForm"},
		},
		{
			name:              "invalid JSON - shell form (not JSON)",
			dockerfileContent: "SHELL /bin/bash -c",
			expectedRules:     []string{"ShellRequiresJsonForm"},
		},
		{
			name:              "invalid JSON - not valid JSON",
			dockerfileContent: `SHELL [/bin/sh, -c]`,
			expectedRules:     []string{"ShellInvalidJsonForm"},
		},
		{
			name:              "invalid JSON - missing closing bracket",
			dockerfileContent: `SHELL ["/bin/sh", "-c"`,
			expectedRules:     []string{"ShellInvalidJsonForm"},
		},
		{
			name:              "invalid JSON - mixed quotes",
			dockerfileContent: `SHELL ["/bin/sh", '-c']`,
			expectedRules:     []string{"ShellInvalidJsonForm"},
		},
		{
			name:              "no arguments",
			dockerfileContent: "SHELL",
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

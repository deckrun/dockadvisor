package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidCOPYFlag(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		expected bool
	}{
		// Valid flags
		{
			name:     "--from flag",
			flag:     "--from",
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
			name:     "--parents flag",
			flag:     "--parents",
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
			name:     "mount flag (not valid for COPY)",
			flag:     "--mount",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCOPYFlag(tt.flag)
			require.Equal(t, tt.expected, result, "isValidCOPYFlag(%q) returned unexpected result", tt.flag)
		})
	}
}

func TestParseCOPY(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid COPY instructions
		{
			name:              "simple copy",
			dockerfileContent: `COPY file.txt /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "multiple source files",
			dockerfileContent: `COPY file1.txt file2.txt /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "wildcard pattern",
			dockerfileContent: `COPY *.txt /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "with --from flag",
			dockerfileContent: `COPY --from=build /app /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "with --chown flag",
			dockerfileContent: `COPY --chown=user:group file.txt /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "with --chmod flag",
			dockerfileContent: `COPY --chmod=755 file.txt /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "with --link flag",
			dockerfileContent: `COPY --link file.txt /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "with multiple flags",
			dockerfileContent: `COPY --from=build --chown=user:group /app /dest/`,
			expectedRules:     []string{},
		},
		{
			name:              "directory copy",
			dockerfileContent: `COPY src/ /dest/`,
			expectedRules:     []string{},
		},
		// Invalid COPY instructions
		{
			name:              "no arguments",
			dockerfileContent: `COPY`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "only source, no destination",
			dockerfileContent: `COPY file.txt`,
			expectedRules:     []string{"CopyMissingArguments"},
		},
		{
			name:              "only flag, no files",
			dockerfileContent: `COPY --from=build`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "invalid flag",
			dockerfileContent: `COPY --invalid file.txt /dest/`,
			expectedRules:     []string{"CopyInvalidFlag"},
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

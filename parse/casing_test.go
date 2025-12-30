package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckConsistentInstructionCasing(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedRuleCodes []string
		expectedCount     int
	}{
		// Valid cases (no violations)
		{
			name: "all uppercase instructions",
			dockerfileContent: `FROM alpine
RUN echo hello
EXPOSE 80
CMD ["sh"]`,
			expectViolation: false,
		},
		{
			name: "all lowercase instructions",
			dockerfileContent: `from alpine
run echo hello
expose 80
cmd ["sh"]`,
			expectViolation: false,
		},
		{
			name:              "single instruction uppercase",
			dockerfileContent: `FROM alpine`,
			expectViolation:   false,
		},
		{
			name:              "single instruction lowercase",
			dockerfileContent: `from alpine`,
			expectViolation:   false,
		},
		// Invalid cases (violations)
		{
			name: "mixed case - majority uppercase",
			dockerfileContent: `FROM alpine
RUN echo hello
from debian
EXPOSE 80`,
			expectViolation:   true,
			expectedRuleCodes: []string{"ConsistentInstructionCasing"},
			expectedCount:     1,
		},
		{
			name: "mixed case - majority lowercase",
			dockerfileContent: `from alpine
run echo hello
FROM debian
expose 80`,
			expectViolation:   true,
			expectedRuleCodes: []string{"ConsistentInstructionCasing"},
			expectedCount:     1,
		},
		{
			name: "pascal case instruction",
			dockerfileContent: `FROM alpine
Run echo hello
EXPOSE 80`,
			expectViolation:   true,
			expectedRuleCodes: []string{"ConsistentInstructionCasing"},
			expectedCount:     1,
		},
		{
			name: "multiple mixed case instructions",
			dockerfileContent: `From alpine
Run echo hello
Expose 80
Cmd ["sh"]`,
			expectViolation: true,
			expectedRuleCodes: []string{
				"ConsistentInstructionCasing",
				"ConsistentInstructionCasing",
				"ConsistentInstructionCasing",
				"ConsistentInstructionCasing",
			},
			expectedCount: 4,
		},
		{
			name: "equal split prefers uppercase",
			dockerfileContent: `FROM alpine
from debian`,
			expectViolation:   true,
			expectedRuleCodes: []string{"ConsistentInstructionCasing"},
			expectedCount:     1,
		},
		{
			name: "camelCase instruction",
			dockerfileContent: `FROM alpine
runCommand echo hello`,
			expectViolation:   true,
			expectedRuleCodes: []string{"ConsistentInstructionCasing", "UnrecognizedInstruction"},
			expectedCount:     2, // Both casing inconsistency and unrecognized instruction
		},
		{
			name: "mixed with all uppercase majority",
			dockerfileContent: `FROM alpine
RUN echo hello
WORKDIR /app
copy . .
EXPOSE 80
CMD ["sh"]`,
			expectViolation:   true,
			expectedRuleCodes: []string{"ConsistentInstructionCasing"},
			expectedCount:     1,
		},
		{
			name: "mixed with all lowercase majority",
			dockerfileContent: `from alpine
run echo hello
workdir /app
COPY . .
expose 80
cmd ["sh"]`,
			expectViolation:   true,
			expectedRuleCodes: []string{"ConsistentInstructionCasing"},
			expectedCount:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			if !tt.expectViolation {
				require.Empty(t, result.Rules, "Expected no violations but got: %v", result.Rules)
			} else {
				require.Len(t, result.Rules, tt.expectedCount,
					"Expected %d violations but got %d: %v", tt.expectedCount, len(result.Rules), result.Rules)

				// Verify all rules have the expected codes
				actualRuleCodes := make([]string, 0, len(result.Rules))
				for _, rule := range result.Rules {
					actualRuleCodes = append(actualRuleCodes, rule.Code)
					require.NotEmpty(t, rule.Description, "Rule description should not be empty")
				}
				require.ElementsMatch(t, tt.expectedRuleCodes, actualRuleCodes,
					"Expected rule codes %v but got %v", tt.expectedRuleCodes, actualRuleCodes)

				// Verify ConsistentInstructionCasing rules have the correct URL
				for _, rule := range result.Rules {
					if rule.Code == "ConsistentInstructionCasing" {
						require.Equal(t, "https://docs.docker.com/reference/build-checks/consistent-instruction-casing/",
							rule.Url, "ConsistentInstructionCasing rule URL should match documentation")
					}
				}
			}
		})
	}
}

func TestParseDockerfileWithConsistentCasing(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRuleCodes []string
	}{
		{
			name: "valid all uppercase",
			dockerfileContent: `FROM alpine
RUN apk add curl
EXPOSE 80`,
			expectedRuleCodes: []string{},
		},
		{
			name: "valid all lowercase",
			dockerfileContent: `from alpine
run apk add curl
expose 80`,
			expectedRuleCodes: []string{},
		},
		{
			name: "invalid mixed casing",
			dockerfileContent: `FROM alpine
run apk add curl
EXPOSE 80`,
			expectedRuleCodes: []string{"ConsistentInstructionCasing"},
		},
		{
			name: "invalid multiple violations",
			dockerfileContent: `From alpine
Run apk add curl
Expose 80`,
			expectedRuleCodes: []string{
				"ConsistentInstructionCasing",
				"ConsistentInstructionCasing",
				"ConsistentInstructionCasing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			if len(tt.expectedRuleCodes) == 0 {
				require.Empty(t, result.Rules, "Expected no rules but got: %v", result.Rules)
			} else {
				require.Len(t, result.Rules, len(tt.expectedRuleCodes),
					"Expected %d rules but got %d: %v", len(tt.expectedRuleCodes), len(result.Rules), result.Rules)

				// Check that all expected rule codes are present
				actualRuleCodes := make([]string, 0, len(result.Rules))
				for _, rule := range result.Rules {
					actualRuleCodes = append(actualRuleCodes, rule.Code)
				}

				require.ElementsMatch(t, tt.expectedRuleCodes, actualRuleCodes,
					"Expected rule codes %v but got %v", tt.expectedRuleCodes, actualRuleCodes)
			}
		})
	}
}

package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnrecognizedInstructions(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
		expectedInstr     string
	}{
		{
			name: "valid dockerfile with all recognized instructions",
			dockerfileContent: `FROM alpine
RUN echo hello
WORKDIR /app
COPY . .
CMD ["echo", "test"]`,
			expectViolation: false,
		},
		{
			name: "unrecognized instruction DOWNLOAD",
			dockerfileContent: `FROM alpine
DOWNLOAD http://example.com/file.tar.gz`,
			expectViolation: true,
			expectedCount:   1,
			expectedInstr:   "DOWNLOAD",
		},
		{
			name: "unrecognized instruction INSTALL",
			dockerfileContent: `FROM alpine
RUN echo hello
INSTALL package.deb`,
			expectViolation: true,
			expectedCount:   1,
			expectedInstr:   "INSTALL",
		},
		{
			name: "multiple unrecognized instructions",
			dockerfileContent: `FROM alpine
DOWNLOAD http://example.com/file.tar.gz
RUN echo hello
INSTALL package.deb
CONFIGURE --with-ssl`,
			expectViolation: true,
			expectedCount:   3,
		},
		{
			name: "lowercase unrecognized instruction",
			dockerfileContent: `FROM alpine
unknown instruction here`,
			expectViolation: true,
			expectedCount:   1,
			expectedInstr:   "unknown",
		},
		{
			name: "typo in instruction name",
			dockerfileContent: `FROM alpine
RUNN echo hello`,
			expectViolation: true,
			expectedCount:   1,
			expectedInstr:   "RUNN",
		},
		{
			name: "misspelled COPY as COYP",
			dockerfileContent: `FROM alpine
COYP . /app`,
			expectViolation: true,
			expectedCount:   1,
			expectedInstr:   "COYP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for UnrecognizedInstruction rules
			unrecognizedRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "UnrecognizedInstruction" {
					unrecognizedRules = append(unrecognizedRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, unrecognizedRules,
					"Expected no UnrecognizedInstruction violations but got: %v", unrecognizedRules)
			} else {
				require.Len(t, unrecognizedRules, tt.expectedCount,
					"Expected %d UnrecognizedInstruction violations but got %d: %v",
					tt.expectedCount, len(unrecognizedRules), unrecognizedRules)

				// Verify rule structure
				for _, rule := range unrecognizedRules {
					require.Equal(t, "UnrecognizedInstruction", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "not a recognized Dockerfile instruction")
					require.Equal(t, "https://docs.docker.com/reference/dockerfile/", rule.Url)

					// If a specific instruction is expected, verify it's in the description
					if tt.expectedInstr != "" && tt.expectedCount == 1 {
						require.Contains(t, rule.Description, tt.expectedInstr,
							"Expected instruction '%s' in description", tt.expectedInstr)
					}
				}
			}
		})
	}
}

func TestUnrecognizedInstructionCaseInsensitive(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedInstr     string
	}{
		{
			name: "uppercase unrecognized",
			dockerfileContent: `FROM alpine
FOOBAR test`,
			expectedInstr: "FOOBAR",
		},
		{
			name: "lowercase unrecognized",
			dockerfileContent: `FROM alpine
foobar test`,
			expectedInstr: "foobar",
		},
		{
			name: "mixed case unrecognized",
			dockerfileContent: `FROM alpine
FooBar test`,
			expectedInstr: "FooBar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Find UnrecognizedInstruction rules
			var found bool
			for _, rule := range result.Rules {
				if rule.Code == "UnrecognizedInstruction" {
					require.Contains(t, rule.Description, tt.expectedInstr,
						"Expected instruction '%s' in description", tt.expectedInstr)
					found = true
				}
			}
			require.True(t, found, "Expected to find UnrecognizedInstruction rule")
		})
	}
}

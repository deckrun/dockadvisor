package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckEmptyContinuations(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		// Valid cases (no violations)
		{
			name: "no continuation lines",
			dockerfileContent: `FROM alpine
RUN echo hello`,
			expectViolation: false,
		},
		{
			name: "continuation with immediate next line",
			dockerfileContent: `FROM alpine
RUN apk add \
    curl`,
			expectViolation: false,
		},
		{
			name: "continuation with comment on next line",
			dockerfileContent: `FROM alpine
RUN apk add \
# This is a comment
    curl`,
			expectViolation: false,
		},
		{
			name: "multiple continuations without empty lines",
			dockerfileContent: `FROM alpine
RUN apk add \
    curl \
    wget \
    vim`,
			expectViolation: false,
		},
		{
			name: "EXPOSE with continuation",
			dockerfileContent: `FROM alpine
EXPOSE \
80`,
			expectViolation: false,
		},
		{
			name: "EXPOSE with comment preventing empty line violation",
			dockerfileContent: `FROM alpine
EXPOSE \
# Port
80`,
			expectViolation: false,
		},
		{
			name: "continuation at end of file",
			dockerfileContent: `FROM alpine
RUN echo hello \`,
			expectViolation: false,
		},
		// Invalid cases (violations)
		{
			name: "empty line after continuation",
			dockerfileContent: `FROM alpine
RUN apk add \

    curl`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "whitespace-only line after continuation",
			dockerfileContent: `FROM alpine
RUN apk add \

    curl`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "multiple empty continuation lines",
			dockerfileContent: `FROM alpine
RUN apk add \

    gnupg \

    curl`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "empty continuation in EXPOSE",
			dockerfileContent: `FROM alpine
EXPOSE \

80`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "empty continuation in LABEL",
			dockerfileContent: `FROM alpine
LABEL version="1.0" \

      maintainer="test@example.com"`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "tabs and spaces on continuation line",
			dockerfileContent: `FROM alpine
RUN apk add \

    curl`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name:              "carriage return on continuation line",
			dockerfileContent: "FROM alpine\nRUN apk add \\\n\r\n    curl",
			expectViolation:   true,
			expectedCount:     1,
		},
		{
			name: "multiple instructions with violations",
			dockerfileContent: `FROM alpine
RUN apk add \

    curl

EXPOSE \

80`,
			expectViolation: true,
			expectedCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := checkEmptyContinuations(tt.dockerfileContent)

			if !tt.expectViolation {
				require.Empty(t, rules, "Expected no violations but got: %v", rules)
			} else {
				require.Len(t, rules, tt.expectedCount, "Expected %d violations but got %d: %v",
					tt.expectedCount, len(rules), rules)

				// Verify all rules have the correct code
				for _, rule := range rules {
					require.Equal(t, "NoEmptyContinuation", rule.Code,
						"Expected rule code 'NoEmptyContinuation' but got '%s'", rule.Code)
					require.NotEmpty(t, rule.Description, "Rule description should not be empty")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/no-empty-continuation/",
						rule.Url, "Rule URL should match documentation")
				}
			}
		})
	}
}

func TestParseDockerfileWithEmptyContinuations(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRuleCodes []string
	}{
		{
			name: "dockerfile with empty continuation",
			dockerfileContent: `FROM alpine
RUN apk add \

    curl`,
			expectedRuleCodes: []string{"NoEmptyContinuation"},
		},
		{
			name: "dockerfile with valid continuation",
			dockerfileContent: `FROM alpine
RUN apk add \
    curl`,
			expectedRuleCodes: []string{},
		},
		{
			name: "dockerfile with continuation and comment",
			dockerfileContent: `FROM alpine
RUN apk add \
# Install curl
    curl`,
			expectedRuleCodes: []string{},
		},
		{
			name: "dockerfile with multiple violations",
			dockerfileContent: `FROM alpine

RUN apk add \

    gnupg \

    curl

EXPOSE \

80`,
			expectedRuleCodes: []string{
				"NoEmptyContinuation",
				"NoEmptyContinuation",
				"NoEmptyContinuation",
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

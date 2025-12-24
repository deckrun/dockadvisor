package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckJSONArgsRecommended(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		// Valid cases - should NOT flag violations
		{
			name: "ENTRYPOINT with exec form",
			dockerfileContent: `FROM alpine
ENTRYPOINT ["echo", "hello"]`,
			expectViolation: false,
		},
		{
			name: "CMD with exec form",
			dockerfileContent: `FROM alpine
CMD ["echo", "hello"]`,
			expectViolation: false,
		},
		{
			name: "shell form with explicit SHELL instruction",
			dockerfileContent: `FROM alpine
RUN apk add bash
SHELL ["/bin/bash", "-c"]
ENTRYPOINT echo "hello world"`,
			expectViolation: false,
		},
		{
			name: "CMD shell form with explicit SHELL instruction",
			dockerfileContent: `FROM alpine
RUN apk add bash
SHELL ["/bin/bash", "-c"]
CMD echo "hello world"`,
			expectViolation: false,
		},
		{
			name: "both CMD and ENTRYPOINT with SHELL instruction",
			dockerfileContent: `FROM alpine
SHELL ["/bin/bash", "-c"]
ENTRYPOINT echo "starting"
CMD echo "running"`,
			expectViolation: false,
		},
		{
			name: "SHELL instruction defined later",
			dockerfileContent: `FROM alpine
ENTRYPOINT echo "hello"
SHELL ["/bin/bash", "-c"]`,
			expectViolation: false,
		},
		{
			name: "wrapper script with heredoc and exec form",
			dockerfileContent: `FROM alpine
RUN apk add bash
COPY --chmod=755 <<EOT /entrypoint.sh
#!/usr/bin/env bash
set -e
my-background-process &
my-program start
EOT
ENTRYPOINT ["/entrypoint.sh"]`,
			expectViolation: false,
		},

		// Invalid cases - SHOULD flag violations
		{
			name: "ENTRYPOINT shell form without SHELL instruction",
			dockerfileContent: `FROM alpine
ENTRYPOINT echo "hello world"`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "CMD shell form without SHELL instruction",
			dockerfileContent: `FROM alpine
CMD echo "hello world"`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "both CMD and ENTRYPOINT shell form without SHELL",
			dockerfileContent: `FROM alpine
ENTRYPOINT /app/start.sh
CMD arg1 arg2`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "CMD with command chaining",
			dockerfileContent: `FROM alpine
CMD apt-get update && apt-get install -y curl`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENTRYPOINT with pipes",
			dockerfileContent: `FROM alpine
ENTRYPOINT cat /data.txt | grep "pattern"`,
			expectViolation: true,
			expectedCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for JSONArgsRecommended rules
			jsonArgsRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "JSONArgsRecommended" {
					jsonArgsRules = append(jsonArgsRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, jsonArgsRules,
					"Expected no JSONArgsRecommended violations but got: %v", jsonArgsRules)
			} else {
				require.Len(t, jsonArgsRules, tt.expectedCount,
					"Expected %d JSONArgsRecommended violations but got %d: %v",
					tt.expectedCount, len(jsonArgsRules), jsonArgsRules)

				// Verify rule structure
				for _, rule := range jsonArgsRules {
					require.Equal(t, "JSONArgsRecommended", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "JSON arguments recommended")
					require.Contains(t, rule.Description, "OS signals")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/json-args-recommended/",
						rule.Url)
				}
			}
		})
	}
}

func TestJSONArgsMultiStage(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		{
			name: "multi-stage with SHELL in one stage",
			dockerfileContent: `FROM alpine AS builder
RUN apk add bash
SHELL ["/bin/bash", "-c"]
ENTRYPOINT echo "building"

FROM alpine AS final
CMD echo "running"`,
			expectViolation: false, // SHELL applies globally
		},
		{
			name: "multi-stage without SHELL",
			dockerfileContent: `FROM alpine AS builder
ENTRYPOINT echo "building"

FROM alpine AS final
CMD echo "running"`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "multi-stage with mixed forms",
			dockerfileContent: `FROM alpine AS builder
ENTRYPOINT ["./build.sh"]

FROM alpine AS final
CMD echo "running"`,
			expectViolation: true,
			expectedCount:   1, // Only CMD should be flagged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for JSONArgsRecommended rules
			jsonArgsRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "JSONArgsRecommended" {
					jsonArgsRules = append(jsonArgsRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, jsonArgsRules,
					"Expected no JSONArgsRecommended violations but got: %v", jsonArgsRules)
			} else {
				require.Len(t, jsonArgsRules, tt.expectedCount,
					"Expected %d JSONArgsRecommended violations but got %d: %v",
					tt.expectedCount, len(jsonArgsRules), jsonArgsRules)
			}
		})
	}
}

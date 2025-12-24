package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckMultipleInstructionsDisallowed(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
		expectedInstType  string // CMD, HEALTHCHECK, or ENTRYPOINT
	}{
		// Valid cases - no violations
		{
			name: "single CMD",
			dockerfileContent: `FROM alpine
CMD ["echo", "hello"]`,
			expectViolation: false,
		},
		{
			name: "single ENTRYPOINT",
			dockerfileContent: `FROM alpine
ENTRYPOINT ["echo", "hello"]`,
			expectViolation: false,
		},
		{
			name: "single HEALTHCHECK",
			dockerfileContent: `FROM alpine
HEALTHCHECK CMD ["curl", "-f", "http://localhost"]`,
			expectViolation: false,
		},
		{
			name: "CMD and ENTRYPOINT together",
			dockerfileContent: `FROM alpine
ENTRYPOINT ["echo"]
CMD ["hello"]`,
			expectViolation: false,
		},
		{
			name: "CMD with HEALTHCHECK (different contexts)",
			dockerfileContent: `FROM alpine
HEALTHCHECK CMD ["curl", "-f", "http://localhost"]
CMD ["python", "-m", "http.server"]`,
			expectViolation: false, // CMD in HEALTHCHECK is separate
		},
		{
			name: "HEALTHCHECK with flags and CMD (documentation example)",
			dockerfileContent: `FROM python:alpine
RUN apk add curl
HEALTHCHECK --interval=1s --timeout=3s \
  CMD ["curl", "-f", "http://localhost:8080"]
CMD ["python", "-m", "http.server", "8080"]`,
			expectViolation: false, // CMD in HEALTHCHECK is separate from container CMD
		},
		{
			name: "multi-stage with one CMD per stage",
			dockerfileContent: `FROM alpine AS builder
CMD ["build"]

FROM alpine AS runtime
CMD ["run"]`,
			expectViolation: false,
		},
		// Invalid cases - violations
		{
			name: "multiple CMD instructions",
			dockerfileContent: `FROM alpine
CMD ["echo", "first"]
CMD ["echo", "second"]`,
			expectViolation:  true,
			expectedCount:    1, // Second CMD is flagged
			expectedInstType: "CMD",
		},
		{
			name: "multiple ENTRYPOINT instructions",
			dockerfileContent: `FROM alpine
ENTRYPOINT ["echo", "Hello, Norway!"]
ENTRYPOINT ["echo", "Hello, Sweden!"]`,
			expectViolation:  true,
			expectedCount:    1, // Second ENTRYPOINT is flagged
			expectedInstType: "ENTRYPOINT",
		},
		{
			name: "multiple HEALTHCHECK instructions",
			dockerfileContent: `FROM alpine
HEALTHCHECK CMD ["curl", "-f", "http://localhost:8080"]
HEALTHCHECK CMD ["curl", "-f", "http://localhost:9090"]`,
			expectViolation:  true,
			expectedCount:    1, // Second HEALTHCHECK is flagged
			expectedInstType: "HEALTHCHECK",
		},
		{
			name: "three CMD instructions",
			dockerfileContent: `FROM alpine
CMD ["echo", "first"]
CMD ["echo", "second"]
CMD ["echo", "third"]`,
			expectViolation:  true,
			expectedCount:    2, // Second and third CMD are flagged
			expectedInstType: "CMD",
		},
		{
			name: "three ENTRYPOINT instructions",
			dockerfileContent: `FROM alpine
ENTRYPOINT ["echo", "first"]
ENTRYPOINT ["echo", "second"]
ENTRYPOINT ["echo", "third"]`,
			expectViolation:  true,
			expectedCount:    2, // Second and third ENTRYPOINT are flagged
			expectedInstType: "ENTRYPOINT",
		},
		{
			name: "multiple CMD in same stage with other instructions",
			dockerfileContent: `FROM alpine
RUN apk add curl
CMD ["echo", "first"]
RUN apk add bash
CMD ["echo", "second"]`,
			expectViolation:  true,
			expectedCount:    1,
			expectedInstType: "CMD",
		},
		// Multi-stage cases
		{
			name: "multiple CMD in first stage only",
			dockerfileContent: `FROM alpine AS builder
CMD ["build", "first"]
CMD ["build", "second"]

FROM alpine AS runtime
CMD ["run"]`,
			expectViolation:  true,
			expectedCount:    1, // Only second CMD in first stage is flagged
			expectedInstType: "CMD",
		},
		{
			name: "multiple CMD in second stage only",
			dockerfileContent: `FROM alpine AS builder
CMD ["build"]

FROM alpine AS runtime
CMD ["run", "first"]
CMD ["run", "second"]`,
			expectViolation:  true,
			expectedCount:    1, // Only second CMD in second stage is flagged
			expectedInstType: "CMD",
		},
		{
			name: "multiple CMD in both stages",
			dockerfileContent: `FROM alpine AS builder
CMD ["build", "first"]
CMD ["build", "second"]

FROM alpine AS runtime
CMD ["run", "first"]
CMD ["run", "second"]`,
			expectViolation:  true,
			expectedCount:    2, // One violation per stage
			expectedInstType: "CMD",
		},
		{
			name: "mixed violations - CMD and ENTRYPOINT",
			dockerfileContent: `FROM alpine
CMD ["echo", "first"]
CMD ["echo", "second"]
ENTRYPOINT ["echo", "first"]
ENTRYPOINT ["echo", "second"]`,
			expectViolation: true,
			expectedCount:   2, // One CMD and one ENTRYPOINT violation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for MultipleInstructionsDisallowed rules
			multipleInstRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "MultipleInstructionsDisallowed" {
					multipleInstRules = append(multipleInstRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, multipleInstRules,
					"Expected no MultipleInstructionsDisallowed violations but got: %v", multipleInstRules)
			} else {
				require.Len(t, multipleInstRules, tt.expectedCount,
					"Expected %d MultipleInstructionsDisallowed violations but got %d: %v",
					tt.expectedCount, len(multipleInstRules), multipleInstRules)

				// Verify rule structure
				for _, rule := range multipleInstRules {
					require.Equal(t, "MultipleInstructionsDisallowed", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "Multiple")
					require.Contains(t, rule.Description, "should not be used in the same stage")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/multiple-instructions-disallowed/",
						rule.Url)

					// If expectedInstType is set, verify it's in the description
					if tt.expectedInstType != "" {
						require.Contains(t, rule.Description, tt.expectedInstType)
					}
				}
			}
		})
	}
}

func TestCheckMultipleInstructionsDisallowedComplexScenarios(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedCount     int
	}{
		{
			name: "all three instructions duplicated",
			dockerfileContent: `FROM alpine
CMD ["first"]
CMD ["second"]
ENTRYPOINT ["first"]
ENTRYPOINT ["second"]
HEALTHCHECK CMD ["first"]
HEALTHCHECK CMD ["second"]`,
			expectedCount: 3, // One violation for each instruction type
		},
		{
			name: "four stages with violations in middle two",
			dockerfileContent: `FROM alpine AS stage1
CMD ["stage1"]

FROM alpine AS stage2
CMD ["stage2-first"]
CMD ["stage2-second"]

FROM alpine AS stage3
CMD ["stage3-first"]
CMD ["stage3-second"]

FROM alpine AS stage4
CMD ["stage4"]`,
			expectedCount: 2, // One violation in stage2, one in stage3
		},
		{
			name: "CMD before first FROM ignored",
			dockerfileContent: `CMD ["ignored"]
FROM alpine
CMD ["first"]
CMD ["second"]`,
			expectedCount: 1, // Only the duplicate in the stage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for MultipleInstructionsDisallowed rules
			multipleInstRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "MultipleInstructionsDisallowed" {
					multipleInstRules = append(multipleInstRules, rule)
				}
			}

			require.Len(t, multipleInstRules, tt.expectedCount,
				"Expected %d MultipleInstructionsDisallowed violations but got %d: %v",
				tt.expectedCount, len(multipleInstRules), multipleInstRules)
		})
	}
}

func TestCheckMultipleInstructionsDisallowedCaseInsensitive(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
	}{
		{
			name: "lowercase cmd instructions",
			dockerfileContent: `FROM alpine
cmd ["first"]
cmd ["second"]`,
			expectViolation: true,
		},
		{
			name: "mixed case CMD instructions",
			dockerfileContent: `FROM alpine
CMD ["first"]
cmd ["second"]`,
			expectViolation: true,
		},
		{
			name: "mixed case ENTRYPOINT instructions",
			dockerfileContent: `FROM alpine
ENTRYPOINT ["first"]
entrypoint ["second"]`,
			expectViolation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for MultipleInstructionsDisallowed rules
			multipleInstRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "MultipleInstructionsDisallowed" {
					multipleInstRules = append(multipleInstRules, rule)
				}
			}

			if tt.expectViolation {
				require.NotEmpty(t, multipleInstRules,
					"Expected MultipleInstructionsDisallowed violation but got none")
			} else {
				require.Empty(t, multipleInstRules,
					"Expected no MultipleInstructionsDisallowed violations but got: %v", multipleInstRules)
			}
		})
	}
}

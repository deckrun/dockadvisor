package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckPlatformFlagConstDisallowed(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		// Valid cases - should NOT flag violations
		{
			name: "multi-stage build with referenced stages",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS build_amd64
RUN apt-get update

FROM --platform=linux/arm64 alpine AS build_arm64
RUN apt-get update

FROM build_${TARGETARCH} AS build
RUN make build`,
			expectViolation: false,
		},
		{
			name: "multi-stage build with exact stage reference",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS builder
RUN echo "building"

FROM builder AS final
RUN echo "final"`,
			expectViolation: false,
		},
		{
			name: "multi-stage with multiple architecture stages referenced",
			dockerfileContent: `FROM --platform=linux/amd64 golang:1.21 AS build_amd64
RUN go build

FROM --platform=linux/arm64 golang:1.21 AS build_arm64
RUN go build

FROM --platform=linux/386 golang:1.21 AS build_386
RUN go build

FROM build_${TARGETARCH}`,
			expectViolation: false,
		},
		{
			name: "platform with variable",
			dockerfileContent: `FROM --platform=$BUILDPLATFORM alpine
RUN echo "building"`,
			expectViolation: false,
		},
		{
			name: "platform with TARGETPLATFORM variable",
			dockerfileContent: `FROM --platform=$TARGETPLATFORM alpine
RUN echo "building"`,
			expectViolation: false,
		},
		{
			name: "no platform flag",
			dockerfileContent: `FROM alpine
RUN echo "building"`,
			expectViolation: false,
		},
		{
			name: "stage with hyphen separator referenced",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS build-amd64
FROM --platform=linux/arm64 alpine AS build-arm64
FROM build-${TARGETARCH}`,
			expectViolation: false,
		},

		// Invalid cases - SHOULD flag violations
		{
			name: "constant platform without stage name",
			dockerfileContent: `FROM --platform=linux/amd64 alpine
RUN echo "building"`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "constant platform with stage name but not referenced",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS builder
RUN echo "building"`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "multiple constant platforms, none referenced",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS builder1
RUN echo "building"

FROM --platform=linux/arm64 alpine AS builder2
RUN echo "building"`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "one referenced, one not",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS builder1
RUN echo "building"

FROM --platform=linux/arm64 alpine AS builder2
RUN echo "building"

FROM builder1
RUN echo "final"`,
			expectViolation: true,
			expectedCount:   1, // Only builder2 should be flagged
		},
		{
			name: "constant platform in final stage",
			dockerfileContent: `FROM alpine AS builder
RUN echo "building"

FROM --platform=linux/amd64 alpine
COPY --from=builder /app /app`,
			expectViolation: true,
			expectedCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for FromPlatformFlagConstDisallowed rules
			platformRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "FromPlatformFlagConstDisallowed" {
					platformRules = append(platformRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, platformRules,
					"Expected no FromPlatformFlagConstDisallowed violations but got: %v", platformRules)
			} else {
				require.Len(t, platformRules, tt.expectedCount,
					"Expected %d FromPlatformFlagConstDisallowed violations but got %d: %v",
					tt.expectedCount, len(platformRules), platformRules)

				// Verify rule structure
				for _, rule := range platformRules {
					require.Equal(t, "FromPlatformFlagConstDisallowed", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "--platform should not use a constant value")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/from-platform-flag-const-disallowed/",
						rule.Url)
				}
			}
		})
	}
}

func TestExtractStagePrefix(t *testing.T) {
	tests := []struct {
		name       string
		stageName  string
		wantPrefix string
	}{
		{
			name:       "underscore separator",
			stageName:  "build_amd64",
			wantPrefix: "build",
		},
		{
			name:       "hyphen separator",
			stageName:  "build-amd64",
			wantPrefix: "build",
		},
		{
			name:       "no separator",
			stageName:  "builder",
			wantPrefix: "builder",
		},
		{
			name:       "multiple underscores",
			stageName:  "build_stage_amd64",
			wantPrefix: "build",
		},
		{
			name:       "multiple hyphens",
			stageName:  "build-stage-amd64",
			wantPrefix: "build",
		},
		{
			name:       "single character",
			stageName:  "b",
			wantPrefix: "b",
		},
		{
			name:       "starts with separator",
			stageName:  "_build",
			wantPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractStagePrefix(tt.stageName)
			require.Equal(t, tt.wantPrefix, got,
				"extractStagePrefix(%q) = %q, want %q", tt.stageName, got, tt.wantPrefix)
		})
	}
}

func TestPlatformConstCaseInsensitive(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
	}{
		{
			name: "uppercase stage name reference",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS BUILD_AMD64
FROM BUILD_AMD64`,
			expectViolation: false,
		},
		{
			name: "mixed case stage name reference",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS Build_Amd64
FROM build_amd64`,
			expectViolation: false,
		},
		{
			name: "lowercase definition uppercase reference",
			dockerfileContent: `FROM --platform=linux/amd64 alpine AS build_amd64
FROM BUILD_AMD64`,
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for FromPlatformFlagConstDisallowed rules
			platformRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "FromPlatformFlagConstDisallowed" {
					platformRules = append(platformRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, platformRules,
					"Expected no FromPlatformFlagConstDisallowed violations but got: %v", platformRules)
			}
		})
	}
}

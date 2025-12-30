package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckInvalidDefaultArgInFrom(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		// Valid cases - no violations
		{
			name: "ARG with default value",
			dockerfileContent: `ARG TAG=latest
FROM busybox:${TAG}`,
			expectViolation: false,
		},
		{
			name: "ARG without default but safe usage",
			dockerfileContent: `ARG VARIANT
FROM busybox:stable${VARIANT}`,
			expectViolation: false,
		},
		{
			name: "ARG without default at image name start",
			dockerfileContent: `ARG PREFIX
FROM ${PREFIX}busybox`,
			expectViolation: false,
		},
		{
			name: "ARG used inside stage not global",
			dockerfileContent: `FROM alpine
ARG TAG
RUN echo ${TAG}`,
			expectViolation: false, // Not a global ARG
		},
		{
			name:              "no ARG usage in FROM",
			dockerfileContent: `FROM alpine:latest`,
			expectViolation:   false,
		},
		{
			name: "multiple ARGs with defaults",
			dockerfileContent: `ARG IMAGE=alpine
ARG TAG=latest
FROM ${IMAGE}:${TAG}`,
			expectViolation: false,
		},
		// Invalid cases - violations
		{
			name: "ARG without default used after colon",
			dockerfileContent: `ARG TAG
FROM busybox:${TAG}`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ARG without default used after @",
			dockerfileContent: `ARG DIGEST
FROM busybox@${DIGEST}`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ARG without default at start before slash",
			dockerfileContent: `ARG REGISTRY
FROM ${REGISTRY}/busybox`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ARG without default in platform flag",
			dockerfileContent: `ARG PLATFORM
FROM --platform=${PLATFORM} alpine`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "multiple ARGs without defaults",
			dockerfileContent: `ARG IMAGE
ARG TAG
FROM ${IMAGE}:${TAG}`,
			expectViolation: true,
			expectedCount:   2, // Both IMAGE and TAG
		},
		{
			name: "mixed ARGs - one with default, one without",
			dockerfileContent: `ARG IMAGE=alpine
ARG TAG
FROM ${IMAGE}:${TAG}`,
			expectViolation: true,
			expectedCount:   1, // Only TAG
		},
		{
			name: "ARG without default used multiple times",
			dockerfileContent: `ARG TAG
FROM busybox:${TAG}
FROM alpine:${TAG}`,
			expectViolation: true,
			expectedCount:   2, // One per FROM usage
		},
		// Edge cases
		{
			name: "ARG with empty default value",
			dockerfileContent: `ARG TAG=
FROM busybox:stable${TAG}`,
			expectViolation: false, // Has a default (even if empty)
		},
		{
			name: "variable at end of image name",
			dockerfileContent: `ARG SUFFIX
FROM busybox${SUFFIX}`,
			expectViolation: false, // busybox with empty suffix is still valid
		},
		{
			name: "variable in middle of tag",
			dockerfileContent: `ARG VERSION
FROM busybox:1.${VERSION}.0`,
			expectViolation: false, // 1..0 is still a valid tag
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for InvalidDefaultArgInFrom rules
			invalidDefaultArgRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "InvalidDefaultArgInFrom" {
					invalidDefaultArgRules = append(invalidDefaultArgRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, invalidDefaultArgRules,
					"Expected no InvalidDefaultArgInFrom violations but got: %v", invalidDefaultArgRules)
			} else {
				require.Len(t, invalidDefaultArgRules, tt.expectedCount,
					"Expected %d InvalidDefaultArgInFrom violations but got %d: %v",
					tt.expectedCount, len(invalidDefaultArgRules), invalidDefaultArgRules)

				// Verify rule structure
				for _, rule := range invalidDefaultArgRules {
					require.Equal(t, "InvalidDefaultArgInFrom", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "no default value")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/invalid-default-arg-in-from/",
						rule.Url)
				}
			}
		})
	}
}

func TestCheckInvalidDefaultArgInFromComplexScenarios(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedCount     int
	}{
		{
			name: "multi-stage with global and stage ARGs",
			dockerfileContent: `ARG BASE_IMAGE
FROM ${BASE_IMAGE}:latest AS builder

ARG BUILD_TAG
FROM alpine:${BUILD_TAG}`,
			expectedCount: 1, // Only BASE_IMAGE (stage ARG BUILD_TAG is not global)
		},
		{
			name: "ARG after FROM not checked",
			dockerfileContent: `FROM alpine
ARG TAG
FROM busybox:${TAG}`,
			expectedCount: 0, // TAG is not a global ARG
		},
		{
			name: "complex image reference with multiple variables",
			dockerfileContent: `ARG REGISTRY=docker.io
ARG ORG
ARG IMAGE
ARG TAG
FROM ${REGISTRY}/${ORG}/${IMAGE}:${TAG}`,
			expectedCount: 3, // ORG, IMAGE, TAG (REGISTRY has default)
		},
		{
			name: "platform and image both use undefined ARGs",
			dockerfileContent: `ARG PLATFORM
ARG IMAGE
FROM --platform=${PLATFORM} ${IMAGE}:latest`,
			expectedCount: 2, // Both PLATFORM and IMAGE
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for InvalidDefaultArgInFrom rules
			invalidDefaultArgRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "InvalidDefaultArgInFrom" {
					invalidDefaultArgRules = append(invalidDefaultArgRules, rule)
				}
			}

			require.Len(t, invalidDefaultArgRules, tt.expectedCount,
				"Expected %d InvalidDefaultArgInFrom violations but got %d: %v",
				tt.expectedCount, len(invalidDefaultArgRules), invalidDefaultArgRules)
		})
	}
}

func TestWouldResultInInvalidImageRef(t *testing.T) {
	tests := []struct {
		name     string
		imageRef string
		varName  string
		expected bool // true if invalid
	}{
		// Invalid patterns
		{"variable after colon", "busybox:${TAG}", "TAG", true},
		{"variable after at", "busybox@${DIGEST}", "DIGEST", true},
		{"variable at start before slash", "${REGISTRY}/busybox", "REGISTRY", true},
		{"variable after colon no braces", "busybox:$TAG", "TAG", true},

		// Valid patterns
		{"variable at end", "busybox${SUFFIX}", "SUFFIX", false},
		{"variable in middle of tag", "busybox:stable${VARIANT}", "VARIANT", false},
		{"variable at start no slash", "${PREFIX}busybox", "PREFIX", false},
		{"variable in middle", "busybox:1.${VERSION}.0", "VERSION", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wouldResultInInvalidImageRef(tt.imageRef, tt.varName)
			require.Equal(t, tt.expected, result,
				"wouldResultInInvalidImageRef(%q, %q) returned unexpected result",
				tt.imageRef, tt.varName)
		})
	}
}

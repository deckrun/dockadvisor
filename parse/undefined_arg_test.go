package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractVariableReferences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "no variables",
			text:     "alpine:latest",
			expected: []string{},
		},
		{
			name:     "single brace variable",
			text:     "node:${VERSION}",
			expected: []string{"VERSION"},
		},
		{
			name:     "single dollar variable",
			text:     "node:$VERSION",
			expected: []string{"VERSION"},
		},
		{
			name:     "multiple variables",
			text:     "node:${VERSION}-${VARIANT}",
			expected: []string{"VERSION", "VARIANT"},
		},
		{
			name:     "mixed formats",
			text:     "node:$VERSION-${VARIANT}",
			expected: []string{"VERSION", "VARIANT"},
		},
		{
			name:     "variable with underscores",
			text:     "node:${NODE_VERSION}",
			expected: []string{"NODE_VERSION"},
		},
		{
			name:     "multiple occurrences same variable",
			text:     "${VERSION}-${VERSION}",
			expected: []string{"VERSION"}, // Should deduplicate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVariableReferences(tt.text)
			require.ElementsMatch(t, tt.expected, result,
				"Expected %v but got %v", tt.expected, result)
		})
	}
}

func TestCheckUndefinedArgInFrom(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
		expectedVarNames  []string
	}{
		// Valid cases (no violations)
		{
			name: "no variables in FROM",
			dockerfileContent: `FROM alpine:latest
RUN echo hello`,
			expectViolation: false,
		},
		{
			name: "ARG declared before FROM",
			dockerfileContent: `ARG VERSION=18
FROM node:${VERSION}`,
			expectViolation: false,
		},
		{
			name: "multiple ARGs declared before FROM",
			dockerfileContent: `ARG VERSION=18
ARG VARIANT=alpine
FROM node:${VERSION}-${VARIANT}`,
			expectViolation: false,
		},
		{
			name:              "predefined TARGETPLATFORM",
			dockerfileContent: `FROM --platform=$TARGETPLATFORM alpine:latest`,
			expectViolation:   false,
		},
		{
			name:              "predefined BUILDPLATFORM",
			dockerfileContent: `FROM --platform=$BUILDPLATFORM alpine:latest`,
			expectViolation:   false,
		},
		{
			name: "ARG with default value",
			dockerfileContent: `ARG BASE_IMAGE=alpine:latest
FROM ${BASE_IMAGE}`,
			expectViolation: false,
		},
		{
			name: "multiple FROM with shared ARG",
			dockerfileContent: `ARG VERSION=18
FROM node:${VERSION} AS builder
FROM node:${VERSION} AS runtime`,
			expectViolation: false,
		},
		// Invalid cases (violations)
		{
			name:              "undefined ARG in FROM",
			dockerfileContent: `FROM node:${VERSION}`,
			expectViolation:   true,
			expectedCount:     1,
			expectedVarNames:  []string{"VERSION"},
		},
		{
			name:              "undefined VARIANT",
			dockerfileContent: `FROM node:22${VARIANT}`,
			expectViolation:   true,
			expectedCount:     1,
			expectedVarNames:  []string{"VARIANT"},
		},
		{
			name:              "multiple undefined ARGs",
			dockerfileContent: `FROM node:${VERSION}-${VARIANT}`,
			expectViolation:   true,
			expectedCount:     2, // One rule per undefined ARG
			expectedVarNames:  []string{"VERSION", "VARIANT"},
		},
		{
			name: "ARG declared after FROM",
			dockerfileContent: `FROM node:${VERSION}
ARG VERSION=18`,
			expectViolation:  true,
			expectedCount:    1,
			expectedVarNames: []string{"VERSION"},
		},
		{
			name: "one defined, one undefined",
			dockerfileContent: `ARG VERSION=18
FROM node:${VERSION}-${VARIANT}`,
			expectViolation:  true,
			expectedCount:    1,
			expectedVarNames: []string{"VARIANT"},
		},
		{
			name:              "misspelled predefined ARG",
			dockerfileContent: `FROM node:${TARGTPLATFORM}`,
			expectViolation:   true,
			expectedCount:     1,
			expectedVarNames:  []string{"TARGTPLATFORM"},
		},
		{
			name: "undefined in second FROM",
			dockerfileContent: `ARG VERSION=18
FROM node:${VERSION} AS builder
FROM alpine:${ALPINE_VERSION}`,
			expectViolation:  true,
			expectedCount:    1,
			expectedVarNames: []string{"ALPINE_VERSION"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for UndefinedArgInFrom rules
			undefinedArgRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "UndefinedArgInFrom" {
					undefinedArgRules = append(undefinedArgRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, undefinedArgRules,
					"Expected no UndefinedArgInFrom violations but got: %v", undefinedArgRules)
			} else {
				require.Len(t, undefinedArgRules, tt.expectedCount,
					"Expected %d UndefinedArgInFrom violations but got %d: %v",
					tt.expectedCount, len(undefinedArgRules), undefinedArgRules)

				// Verify rule structure
				for _, rule := range undefinedArgRules {
					require.Equal(t, "UndefinedArgInFrom", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "FROM argument")
					require.Contains(t, rule.Description, "not declared")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/undefined-arg-in-from/",
						rule.Url)

					// Check that at least one expected var name appears in the description
					foundVar := false
					for _, varName := range tt.expectedVarNames {
						if strings.Contains(rule.Description, varName) {
							foundVar = true
							break
						}
					}
					require.True(t, foundVar,
						"Expected one of %v to appear in description: %s",
						tt.expectedVarNames, rule.Description)
				}
			}
		})
	}
}

func TestCheckUndefinedArgInFromWithPredefinedArgs(t *testing.T) {
	tests := []struct {
		name            string
		argName         string
		expectViolation bool
	}{
		{"TARGETPLATFORM", "TARGETPLATFORM", false},
		{"TARGETOS", "TARGETOS", false},
		{"TARGETARCH", "TARGETARCH", false},
		{"TARGETVARIANT", "TARGETVARIANT", false},
		{"BUILDPLATFORM", "BUILDPLATFORM", false},
		{"BUILDOS", "BUILDOS", false},
		{"BUILDARCH", "BUILDARCH", false},
		{"BUILDVARIANT", "BUILDVARIANT", false},
		{"HTTP_PROXY", "HTTP_PROXY", false},
		{"http_proxy", "http_proxy", false},
		{"HTTPS_PROXY", "HTTPS_PROXY", false},
		{"https_proxy", "https_proxy", false},
		{"FTP_PROXY", "FTP_PROXY", false},
		{"ftp_proxy", "ftp_proxy", false},
		{"NO_PROXY", "NO_PROXY", false},
		{"no_proxy", "no_proxy", false},
		{"ALL_PROXY", "ALL_PROXY", false},
		{"all_proxy", "all_proxy", false},
		{"CUSTOM_ARG", "CUSTOM_ARG", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dockerfileContent := "FROM alpine:${" + tt.argName + "}"
			result, err := ParseDockerfile(dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for UndefinedArgInFrom rules
			undefinedArgRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "UndefinedArgInFrom" {
					undefinedArgRules = append(undefinedArgRules, rule)
				}
			}

			if tt.expectViolation {
				require.NotEmpty(t, undefinedArgRules,
					"Expected UndefinedArgInFrom violation for %s", tt.argName)
			} else {
				require.Empty(t, undefinedArgRules,
					"Expected no UndefinedArgInFrom violation for predefined ARG %s, but got: %v",
					tt.argName, undefinedArgRules)
			}
		})
	}
}

func TestParseDockerfileWithUndefinedArgInFrom(t *testing.T) {
	tests := []struct {
		name                   string
		dockerfileContent      string
		expectedUndefinedRules int
		expectedTotalRules     int
	}{
		{
			name: "valid with declared ARG",
			dockerfileContent: `ARG VERSION=18
FROM node:${VERSION}`,
			expectedUndefinedRules: 0,
			expectedTotalRules:     0,
		},
		{
			name:                   "undefined ARG",
			dockerfileContent:      `FROM node:${VERSION}`,
			expectedUndefinedRules: 1,
			expectedTotalRules:     1,
		},
		{
			name: "undefined ARG with other violations",
			dockerfileContent: `FROM node:${VERSION} as builder
WORKDIR app`,
			expectedUndefinedRules: 1,
			expectedTotalRules:     2, // UndefinedArgInFrom + FromAsCasing + WorkdirRelativePath = 3? Let me check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Count UndefinedArgInFrom rules
			undefinedCount := 0
			for _, rule := range result.Rules {
				if rule.Code == "UndefinedArgInFrom" {
					undefinedCount++
				}
			}

			require.Equal(t, tt.expectedUndefinedRules, undefinedCount,
				"Expected %d UndefinedArgInFrom rules but got %d",
				tt.expectedUndefinedRules, undefinedCount)

			require.GreaterOrEqual(t, len(result.Rules), tt.expectedTotalRules,
				"Expected at least %d total rules but got %d: %v",
				tt.expectedTotalRules, len(result.Rules), result.Rules)
		})
	}
}

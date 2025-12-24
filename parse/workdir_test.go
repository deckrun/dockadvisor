package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWORKDIR(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid WORKDIR instructions - Unix absolute paths
		{
			name:              "absolute path with root",
			dockerfileContent: `WORKDIR /`,
			expectedRules:     []string{},
		},
		{
			name:              "absolute path standard",
			dockerfileContent: `WORKDIR /usr/share/nginx/html`,
			expectedRules:     []string{},
		},
		{
			name:              "absolute path simple",
			dockerfileContent: `WORKDIR /app`,
			expectedRules:     []string{},
		},
		// Valid WORKDIR instructions - Windows absolute paths
		{
			name:              "Windows path with backslash",
			dockerfileContent: `WORKDIR C:\app`,
			expectedRules:     []string{},
		},
		{
			name:              "Windows path with forward slash",
			dockerfileContent: `WORKDIR C:/app`,
			expectedRules:     []string{},
		},
		{
			name:              "Windows path lowercase drive",
			dockerfileContent: `WORKDIR c:/program files/app`,
			expectedRules:     []string{},
		},
		// Valid WORKDIR instructions - Variable references
		{
			name:              "variable with braces",
			dockerfileContent: `WORKDIR ${WORKDIR}`,
			expectedRules:     []string{},
		},
		{
			name:              "variable without braces",
			dockerfileContent: `WORKDIR $HOME`,
			expectedRules:     []string{},
		},
		{
			name:              "variable with path",
			dockerfileContent: `WORKDIR $HOME/app`,
			expectedRules:     []string{},
		},
		{
			name:              "variable with braces and path",
			dockerfileContent: `WORKDIR ${BUILD_PATH}/dist`,
			expectedRules:     []string{},
		},
		// Invalid WORKDIR instructions
		{
			name:              "relative path with dot",
			dockerfileContent: `WORKDIR ./build`,
			expectedRules:     []string{"WorkdirRelativePath"},
		},
		{
			name:              "relative path without dot",
			dockerfileContent: `WORKDIR usr/share/nginx/html`,
			expectedRules:     []string{"WorkdirRelativePath"},
		},
		{
			name: "nginx example with relative WORKDIR (bad)",
			dockerfileContent: `FROM nginx AS web
WORKDIR usr/share/nginx/html
COPY public .`,
			expectedRules: []string{"WorkdirRelativePath"},
		},
		{
			name: "nginx example with absolute WORKDIR (good)",
			dockerfileContent: `FROM nginx AS web
WORKDIR /usr/share/nginx/html
COPY public .`,
			expectedRules: []string{},
		},
		{
			name:              "relative path single directory",
			dockerfileContent: `WORKDIR build`,
			expectedRules:     []string{"WorkdirRelativePath"},
		},
		{
			name:              "relative path with double dot",
			dockerfileContent: `WORKDIR ../build`,
			expectedRules:     []string{"WorkdirRelativePath"},
		},
		{
			name:              "no arguments",
			dockerfileContent: `WORKDIR`,
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

func TestCheckWorkdirAbsolute(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Unix absolute paths
		{
			name:     "absolute path with single slash",
			path:     "/",
			expected: true,
		},
		{
			name:     "absolute path",
			path:     "/usr/share/nginx/html",
			expected: true,
		},
		{
			name:     "absolute path with app directory",
			path:     "/app",
			expected: true,
		},
		// Windows absolute paths
		{
			name:     "Windows path with backslash",
			path:     `C:\app`,
			expected: true,
		},
		{
			name:     "Windows path with forward slash",
			path:     "C:/app",
			expected: true,
		},
		{
			name:     "Windows path drive only",
			path:     "C:",
			expected: true,
		},
		{
			name:     "Windows path lowercase drive",
			path:     "c:/program files/app",
			expected: true,
		},
		{
			name:     "Windows path D drive",
			path:     "D:/data",
			expected: true,
		},
		// Variable references (treated as absolute)
		{
			name:     "variable with braces",
			path:     "${WORKDIR}",
			expected: true,
		},
		{
			name:     "variable without braces",
			path:     "$HOME",
			expected: true,
		},
		{
			name:     "variable with path",
			path:     "$HOME/app",
			expected: true,
		},
		{
			name:     "variable with braces and path",
			path:     "${BUILD_PATH}/dist",
			expected: true,
		},
		{
			name:     "variable APP_HOME",
			path:     "$APP_HOME",
			expected: true,
		},
		// Relative paths
		{
			name:     "relative path with dot",
			path:     "./build",
			expected: false,
		},
		{
			name:     "relative path without dot",
			path:     "usr/share/nginx/html",
			expected: false,
		},
		{
			name:     "relative path single directory",
			path:     "build",
			expected: false,
		},
		{
			name:     "relative path with double dot",
			path:     "../build",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkWorkdirAbsolute(tt.path)
			require.Equal(t, tt.expected, result, "checkWorkdirAbsolute(%q) returned unexpected result", tt.path)
		})
	}
}

package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckUSERFormat(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		// Valid formats
		{
			name:     "simple username",
			config:   "patrick",
			expected: true,
		},
		{
			name:     "username with underscore",
			config:   "app_user",
			expected: true,
		},
		{
			name:     "username with dash",
			config:   "app-user",
			expected: true,
		},
		{
			name:     "username with dot",
			config:   "app.user",
			expected: true,
		},
		{
			name:     "UID numeric",
			config:   "1000",
			expected: true,
		},
		{
			name:     "username with group",
			config:   "patrick:developers",
			expected: true,
		},
		{
			name:     "UID with GID",
			config:   "1000:1000",
			expected: true,
		},
		{
			name:     "username with GID",
			config:   "patrick:1000",
			expected: true,
		},
		{
			name:     "UID with group name",
			config:   "1000:developers",
			expected: true,
		},
		{
			name:     "complex username",
			config:   "app_user-01",
			expected: true,
		},
		{
			name:     "variable syntax",
			config:   "$USER_NAME",
			expected: true,
		},
		{
			name:     "username with variable group",
			config:   "patrick:$GROUP_NAME",
			expected: true,
		},
		// Invalid formats
		{
			name:     "empty string",
			config:   "",
			expected: false,
		},
		{
			name:     "whitespace only",
			config:   "   ",
			expected: false,
		},
		{
			name:     "multiple colons",
			config:   "user:group:extra",
			expected: false,
		},
		{
			name:     "starts with colon",
			config:   ":group",
			expected: false,
		},
		{
			name:     "ends with colon",
			config:   "user:",
			expected: false,
		},
		{
			name:     "contains spaces",
			config:   "user name",
			expected: false,
		},
		{
			name:     "special characters",
			config:   "user@host",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkUSERFormat(tt.config)
			require.Equal(t, tt.expected, result, "checkUSERFormat(%q) returned unexpected result", tt.config)
		})
	}
}

func TestParseUSER(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid USER instructions
		{
			name:              "simple username",
			dockerfileContent: `USER patrick`,
			expectedRules:     []string{},
		},
		{
			name:              "username with underscore",
			dockerfileContent: `USER app_user`,
			expectedRules:     []string{},
		},
		{
			name:              "username with dash",
			dockerfileContent: `USER app-user`,
			expectedRules:     []string{},
		},
		{
			name:              "UID numeric",
			dockerfileContent: `USER 1000`,
			expectedRules:     []string{},
		},
		{
			name:              "username with group",
			dockerfileContent: `USER patrick:developers`,
			expectedRules:     []string{},
		},
		{
			name:              "UID with GID",
			dockerfileContent: `USER 1000:1000`,
			expectedRules:     []string{},
		},
		{
			name:              "username with GID",
			dockerfileContent: `USER patrick:1000`,
			expectedRules:     []string{},
		},
		{
			name:              "UID with group name",
			dockerfileContent: `USER 1000:developers`,
			expectedRules:     []string{},
		},
		{
			name:              "variable syntax",
			dockerfileContent: `USER $USER_NAME`,
			expectedRules:     []string{},
		},
		{
			name:              "username with variable group",
			dockerfileContent: `USER patrick:$GROUP_NAME`,
			expectedRules:     []string{},
		},
		// Invalid USER instructions
		{
			name:              "no arguments",
			dockerfileContent: `USER`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "multiple colons",
			dockerfileContent: `USER user:group:extra`,
			expectedRules:     []string{"UserInvalidFormat"},
		},
		{
			name:              "starts with colon",
			dockerfileContent: `USER :group`,
			expectedRules:     []string{"UserInvalidFormat"},
		},
		{
			name:              "ends with colon",
			dockerfileContent: `USER user:`,
			expectedRules:     []string{"UserInvalidFormat"},
		},
		{
			name:              "contains spaces",
			dockerfileContent: `USER user name`,
			expectedRules:     []string{"UserInvalidFormat"},
		},
		{
			name:              "special characters",
			dockerfileContent: `USER user@host`,
			expectedRules:     []string{"UserInvalidFormat"},
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

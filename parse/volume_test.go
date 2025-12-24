package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckVOLUMEJsonForm(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected bool
	}{
		// Valid JSON forms
		{
			name:     "single volume JSON array",
			config:   `["/data"]`,
			expected: true,
		},
		{
			name:     "multiple volumes JSON array",
			config:   `["/var/log", "/var/db"]`,
			expected: true,
		},
		{
			name:     "complex path with spaces",
			config:   `["/my data"]`,
			expected: true,
		},
		{
			name:     "Windows-style path",
			config:   `["C:\\data"]`,
			expected: true,
		},
		// Invalid JSON forms
		{
			name:     "empty array",
			config:   `[]`,
			expected: false,
		},
		{
			name:     "single quotes instead of double",
			config:   `['/data']`,
			expected: false,
		},
		{
			name:     "missing closing bracket",
			config:   `["/data"`,
			expected: false,
		},
		{
			name:     "not valid JSON",
			config:   `[/data]`,
			expected: false,
		},
		{
			name:     "mixed quotes",
			config:   `["/data", '/var/log']`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkVOLUMEJsonForm(tt.config)
			require.Equal(t, tt.expected, result, "checkVOLUMEJsonForm(%q) returned unexpected result", tt.config)
		})
	}
}

func TestParseVOLUME(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid VOLUME instructions - shell form
		{
			name:              "single volume shell form",
			dockerfileContent: `VOLUME /data`,
			expectedRules:     []string{},
		},
		{
			name:              "multiple volumes shell form",
			dockerfileContent: `VOLUME /var/log /var/db`,
			expectedRules:     []string{},
		},
		// Valid VOLUME instructions - JSON form
		{
			name:              "single volume JSON form",
			dockerfileContent: `VOLUME ["/data"]`,
			expectedRules:     []string{},
		},
		{
			name:              "multiple volumes JSON form",
			dockerfileContent: `VOLUME ["/var/log", "/var/db"]`,
			expectedRules:     []string{},
		},
		{
			name:              "Windows path JSON form",
			dockerfileContent: `VOLUME ["C:\\data"]`,
			expectedRules:     []string{},
		},
		// Invalid VOLUME instructions
		{
			name:              "no arguments",
			dockerfileContent: `VOLUME`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "empty JSON array",
			dockerfileContent: `VOLUME []`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "invalid JSON - single quotes",
			dockerfileContent: `VOLUME ['/data']`,
			expectedRules:     []string{"VolumeInvalidJsonForm"},
		},
		{
			name:              "invalid JSON - not valid JSON",
			dockerfileContent: `VOLUME [/data]`,
			expectedRules:     []string{"VolumeInvalidJsonForm"},
		},
		{
			name:              "invalid JSON - missing closing bracket",
			dockerfileContent: `VOLUME ["/data"`,
			expectedRules:     []string{"VolumeInvalidJsonForm"},
		},
		{
			name:              "invalid JSON - mixed quotes",
			dockerfileContent: `VOLUME ["/data", '/var/log']`,
			expectedRules:     []string{"VolumeInvalidJsonForm"},
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

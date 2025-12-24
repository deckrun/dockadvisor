package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSTOPSIGNAL(t *testing.T) {
	tests := []struct {
		name          string
		dockerfile    string
		expectedRules []string
	}{
		// Valid STOPSIGNAL instructions
		{
			name:          "signal name",
			dockerfile:    `FROM alpine\nSTOPSIGNAL SIGTERM`,
			expectedRules: []string{},
		},
		{
			name:          "signal number",
			dockerfile:    `FROM alpine\nSTOPSIGNAL 9`,
			expectedRules: []string{},
		},
		{
			name:          "SIGKILL",
			dockerfile:    `FROM alpine\nSTOPSIGNAL SIGKILL`,
			expectedRules: []string{},
		},
		// Invalid STOPSIGNAL instructions
		{
			name: "no arguments",
			dockerfile: `FROM alpine
STOPSIGNAL`,
			expectedRules: []string{"InvalidInstruction"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfile)
			require.NoError(t, err)

			actualRuleCodes := make(map[string]bool)
			for _, rule := range result.Rules {
				actualRuleCodes[rule.Code] = true
			}

			for _, expectedCode := range tt.expectedRules {
				require.True(t, actualRuleCodes[expectedCode], "Expected rule code %q not found in results. Got rules: %v", expectedCode, result.Rules)
			}

			if len(tt.expectedRules) == 0 {
				for _, rule := range result.Rules {
					if strings.HasPrefix(rule.Code, "Stopsignal") {
						t.Errorf("Expected no STOPSIGNAL rules but got: %v", rule)
					}
				}
			}
		})
	}
}

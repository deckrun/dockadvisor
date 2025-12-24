package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHEALTHCHECK(t *testing.T) {
	tests := []struct {
		name          string
		dockerfile    string
		expectedRules []string
	}{
		// Valid HEALTHCHECK instructions
		{
			name:          "healthcheck with CMD",
			dockerfile:    `FROM alpine\nHEALTHCHECK CMD /bin/check`,
			expectedRules: []string{},
		},
		{
			name:          "healthcheck with options and CMD",
			dockerfile:    `FROM alpine\nHEALTHCHECK --interval=5m --timeout=3s CMD curl -f http://localhost/`,
			expectedRules: []string{},
		},
		{
			name:          "healthcheck NONE",
			dockerfile:    `FROM alpine\nHEALTHCHECK NONE`,
			expectedRules: []string{},
		},
		// Invalid HEALTHCHECK instructions
		{
			name: "no arguments",
			dockerfile: `FROM alpine
HEALTHCHECK`,
			expectedRules: []string{"InvalidInstruction"},
		},
		{
			name: "missing CMD keyword",
			dockerfile: `FROM alpine
HEALTHCHECK /bin/check`,
			expectedRules: []string{"HealthcheckMissingCmd"},
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
					if strings.HasPrefix(rule.Code, "Healthcheck") {
						t.Errorf("Expected no HEALTHCHECK rules but got: %v", rule)
					}
				}
			}
		})
	}
}

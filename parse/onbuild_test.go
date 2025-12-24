package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseONBUILD(t *testing.T) {
	tests := []struct {
		name          string
		dockerfile    string
		expectedRules []string
	}{
		// Valid ONBUILD instructions
		{
			name:          "onbuild with RUN",
			dockerfile:    `ONBUILD RUN echo "building"`,
			expectedRules: []string{},
		},
		{
			name:          "onbuild with ADD",
			dockerfile:    `ONBUILD ADD . /app`,
			expectedRules: []string{},
		},
		{
			name:          "onbuild with COPY",
			dockerfile:    `ONBUILD COPY requirements.txt /app/`,
			expectedRules: []string{},
		},
		// Invalid ONBUILD instructions
		{
			name:          "no arguments",
			dockerfile:    `ONBUILD`,
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
					if strings.HasPrefix(rule.Code, "Onbuild") {
						t.Errorf("Expected no ONBUILD rules but got: %v", rule)
					}
				}
			}
		})
	}
}

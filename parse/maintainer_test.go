package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMAINTAINER(t *testing.T) {
	tests := []struct {
		name          string
		dockerfile    string
		expectedRules []string
	}{
		// Valid but deprecated MAINTAINER instructions
		{
			name: "maintainer with email",
			dockerfile: `FROM alpine
MAINTAINER user@example.com`,
			expectedRules: []string{"MaintainerDeprecated"},
		},
		{
			name: "maintainer with name and email",
			dockerfile: `FROM alpine
MAINTAINER John Doe <john@example.com>`,
			expectedRules: []string{"MaintainerDeprecated"},
		},
		// Invalid MAINTAINER instructions
		{
			name: "no arguments",
			dockerfile: `FROM alpine
MAINTAINER`,
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

			// For MAINTAINER, we always expect the deprecated warning
			if len(tt.expectedRules) > 0 && !strings.Contains(tt.name, "no arguments") {
				hasDeprecatedWarning := false
				for _, rule := range result.Rules {
					if rule.Code == "MaintainerDeprecated" {
						hasDeprecatedWarning = true
						break
					}
				}
				require.True(t, hasDeprecatedWarning, "Expected MaintainerDeprecated warning")
			}
		})
	}
}

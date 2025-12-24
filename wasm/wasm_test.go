//go:build js && wasm

package main

import (
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWASMParseDockerfile(t *testing.T) {
	tests := []struct {
		name               string
		dockerfileContent  string
		expectSuccess      bool
		expectedRulesCount int
	}{
		{
			name: "valid dockerfile with no issues",
			dockerfileContent: `FROM ubuntu:20.04
WORKDIR /app
RUN echo "hello"`,
			expectSuccess:      true,
			expectedRulesCount: 0,
		},
		{
			name: "dockerfile with relative WORKDIR",
			dockerfileContent: `FROM ubuntu:20.04
WORKDIR app
RUN echo "hello"`,
			expectSuccess:      true,
			expectedRulesCount: 1, // Should trigger WorkdirRelativePath rule
		},
		{
			name: "dockerfile with multiple rules",
			dockerfileContent: `FROM ubuntu:20.04
WORKDIR app
EXPOSE 80:80
RUN echo "hello"`,
			expectSuccess:      true,
			expectedRulesCount: 2, // Should trigger WorkdirRelativePath and ExposeFormat rules
		},
		{
			name: "a more complex Dockerfile",
			dockerfileContent: `# builder step builds the image
FROM golang:1.21 as builder
WORKDIR /build
COPY . .
RUN go build -o app

# Runtime stage
FROM alpine:latest
WORKDIR relative/path
COPY --from=builder /build/app /app
RUN chmod +x /app
CMD ["/app"]`,
			expectSuccess:      true,
			expectedRulesCount: 2, // Should trigger FromAsCasing and WorkdirRelativePath rule
		},
		{
			name:              "empty dockerfile",
			dockerfileContent: "",
			expectSuccess:     false, // Parser will error on empty file
		},
		{
			name: "dockerfile with only comments",
			dockerfileContent: `# This is a comment
# Another comment`,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []js.Value{
				js.ValueOf(tt.dockerfileContent),
			}
			resultAny := parseDockerfile(js.Undefined(), args)
			result, ok := resultAny.(map[string]any)
			require.True(t, ok, "expected resultAny to be map[string]any")

			success, ok := result["success"].(bool)
			require.True(t, ok, "expected success field to be bool")
			require.Equal(t, tt.expectSuccess, success, "unexpected success value")

			if tt.expectSuccess {
				rulesSliceOfAny, ok := result["rules"].([]any)
				require.True(t, ok, "expected rules field to be []any")

				if tt.expectedRulesCount == 0 {
					require.Empty(t, rulesSliceOfAny, "expected no rules for successful parse")
				} else {
					require.Len(t, rulesSliceOfAny, tt.expectedRulesCount, "unexpected number of rules")
					for _, ruleAny := range rulesSliceOfAny {
						rule, ok := ruleAny.(map[string]any)
						require.True(t, ok, "expected each rule to be map[string]any")
						require.Contains(t, rule, "startLine", "rule should contain StartLine field")
						require.Contains(t, rule, "endLine", "rule should contain EndLine field")
						require.Contains(t, rule, "code", "rule should contain Code field")
						require.Contains(t, rule, "description", "rule should contain Description field")

						// Check that all expected fields exist and have correct types
						if startLine, ok := rule["startLine"].(int); !ok {
							t.Errorf("startLine should be int, got %T", rule["startLine"])
						} else {
							require.Greater(t, startLine, 0, "startLine should be > 0")
						}

						if endLine, ok := rule["endLine"].(int); !ok {
							t.Errorf("endLine should be int, got %T", rule["endLine"])
						} else {
							require.Greater(t, endLine, 0, "endLine should be > 0")
						}

						if code, ok := rule["code"].(string); !ok {
							t.Errorf("code should be string, got %T", rule["code"])
						} else {
							require.NotEmpty(t, code, "code should not be empty")
						}

						if description, ok := rule["description"].(string); !ok {
							t.Errorf("description should be string, got %T", rule["description"])
						} else {
							require.NotEmpty(t, description, "description should not be empty")
						}
					}
				}
			}
		})
	}
}

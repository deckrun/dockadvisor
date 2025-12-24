package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckExecFormJSON(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		// Valid exec forms
		{
			name:     "simple exec form",
			command:  `["echo", "hello"]`,
			expected: true,
		},
		{
			name:     "exec form with single command",
			command:  `["/bin/bash"]`,
			expected: true,
		},
		{
			name:     "exec form with multiple args",
			command:  `["apt-get", "install", "-y", "curl"]`,
			expected: true,
		},
		{
			name:     "exec form with spaces",
			command:  `  ["echo", "hello"]  `,
			expected: true,
		},
		{
			name:     "exec form with escaped characters",
			command:  `["c:\\windows\\system32\\tasklist.exe"]`,
			expected: true,
		},
		// Invalid exec forms
		{
			name:     "empty array",
			command:  `[]`,
			expected: false,
		},
		{
			name:     "single quotes instead of double",
			command:  `['echo', 'hello']`,
			expected: false,
		},
		{
			name:     "missing closing bracket",
			command:  `["echo", "hello"`,
			expected: false,
		},
		{
			name:     "missing opening bracket",
			command:  `"echo", "hello"]`,
			expected: false,
		},
		{
			name:     "not valid JSON",
			command:  `[echo, hello]`,
			expected: false,
		},
		{
			name:     "mixed quotes",
			command:  `["echo", 'hello']`,
			expected: false,
		},
		{
			name:     "just a string",
			command:  `echo hello`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkExecFormJSON(tt.command)
			require.Equal(t, tt.expected, result, "checkExecFormJSON(%q) returned unexpected result", tt.command)
		})
	}
}

func TestCheckMountFlag(t *testing.T) {
	tests := []struct {
		name       string
		mountValue string
		expected   bool
	}{
		// Valid mount flags
		{
			name:       "type=bind",
			mountValue: "type=bind",
			expected:   true,
		},
		{
			name:       "type=cache with options",
			mountValue: "type=cache,target=/root/.cache/go-build",
			expected:   true,
		},
		{
			name:       "type=tmpfs",
			mountValue: "type=tmpfs,target=/tmp",
			expected:   true,
		},
		{
			name:       "type=secret",
			mountValue: "type=secret,id=aws,target=/root/.aws/credentials",
			expected:   true,
		},
		{
			name:       "type=ssh",
			mountValue: "type=ssh",
			expected:   true,
		},
		{
			name:       "no type specified (defaults to bind)",
			mountValue: "target=/app",
			expected:   true,
		},
		// Invalid mount flags
		{
			name:       "empty value",
			mountValue: "",
			expected:   false,
		},
		{
			name:       "invalid type",
			mountValue: "type=invalid",
			expected:   false,
		},
		{
			name:       "type=unknown with options",
			mountValue: "type=unknown,target=/tmp",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkMountFlag(tt.mountValue)
			require.Equal(t, tt.expected, result, "checkMountFlag(%q) returned unexpected result", tt.mountValue)
		})
	}
}

func TestCheckNetworkFlag(t *testing.T) {
	tests := []struct {
		name         string
		networkValue string
		expected     bool
	}{
		// Valid network values
		{
			name:         "default",
			networkValue: "default",
			expected:     true,
		},
		{
			name:         "none",
			networkValue: "none",
			expected:     true,
		},
		{
			name:         "host",
			networkValue: "host",
			expected:     true,
		},
		// Invalid network values
		{
			name:         "invalid value",
			networkValue: "invalid",
			expected:     false,
		},
		{
			name:         "empty string",
			networkValue: "",
			expected:     false,
		},
		{
			name:         "uppercase DEFAULT",
			networkValue: "DEFAULT",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkNetworkFlag(tt.networkValue)
			require.Equal(t, tt.expected, result, "checkNetworkFlag(%q) returned unexpected result", tt.networkValue)
		})
	}
}

func TestCheckSecurityFlag(t *testing.T) {
	tests := []struct {
		name          string
		securityValue string
		expected      bool
	}{
		// Valid security values
		{
			name:          "sandbox",
			securityValue: "sandbox",
			expected:      true,
		},
		{
			name:          "insecure",
			securityValue: "insecure",
			expected:      true,
		},
		// Invalid security values
		{
			name:          "invalid value",
			securityValue: "invalid",
			expected:      false,
		},
		{
			name:          "empty string",
			securityValue: "",
			expected:      false,
		},
		{
			name:          "uppercase SANDBOX",
			securityValue: "SANDBOX",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSecurityFlag(tt.securityValue)
			require.Equal(t, tt.expected, result, "checkSecurityFlag(%q) returned unexpected result", tt.securityValue)
		})
	}
}

func TestParseRUN(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid RUN instructions
		{
			name:              "simple shell form",
			dockerfileContent: `RUN echo hello`,
			expectedRules:     []string{},
		},
		{
			name:              "shell form with multiple commands",
			dockerfileContent: `RUN apt-get update && apt-get install -y curl`,
			expectedRules:     []string{},
		},
		{
			name:              "valid exec form",
			dockerfileContent: `RUN ["echo", "hello"]`,
			expectedRules:     []string{},
		},
		{
			name:              "exec form with multiple args",
			dockerfileContent: `RUN ["/bin/sh", "-c", "echo hello"]`,
			expectedRules:     []string{},
		},
		{
			name:              "with --mount flag",
			dockerfileContent: `RUN --mount=type=cache,target=/root/.cache/go-build go build`,
			expectedRules:     []string{},
		},
		{
			name:              "with --network=none",
			dockerfileContent: `RUN --network=none pip install mypackage`,
			expectedRules:     []string{},
		},
		{
			name:              "with --network=host",
			dockerfileContent: `RUN --network=host apk add curl`,
			expectedRules:     []string{},
		},
		{
			name:              "with --security=insecure",
			dockerfileContent: `RUN --security=insecure some-privileged-command`,
			expectedRules:     []string{},
		},
		{
			name:              "with multiple valid flags",
			dockerfileContent: `RUN --mount=type=cache,target=/go/pkg --network=none go build`,
			expectedRules:     []string{},
		},
		{
			name: "shell form with heredoc",
			dockerfileContent: `RUN <<EOF
apt-get update
apt-get install -y curl
EOF`,
			expectedRules: []string{},
		},
		// Invalid RUN instructions
		{
			name:              "invalid exec form with single quotes",
			dockerfileContent: `RUN ['echo', 'hello']`,
			expectedRules:     []string{"RunInvalidExecForm"},
		},
		{
			name:              "invalid exec form - not valid JSON",
			dockerfileContent: `RUN [echo, hello]`,
			expectedRules:     []string{"RunInvalidExecForm"},
		},
		{
			name:              "invalid exec form - empty array",
			dockerfileContent: `RUN []`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		{
			name:              "invalid --network flag",
			dockerfileContent: `RUN --network=invalid echo hello`,
			expectedRules:     []string{"RunInvalidNetworkFlag"},
		},
		{
			name:              "invalid --security flag",
			dockerfileContent: `RUN --security=invalid echo hello`,
			expectedRules:     []string{"RunInvalidSecurityFlag"},
		},
		{
			name:              "invalid --mount type",
			dockerfileContent: `RUN --mount=type=invalid,target=/tmp echo hello`,
			expectedRules:     []string{"RunInvalidMountFlag"},
		},
		// Edge cases
		{
			name:              "exec form with escaped backslashes",
			dockerfileContent: `RUN ["c:\\windows\\system32\\tasklist.exe"]`,
			expectedRules:     []string{},
		},
		{
			name:              "mount without type (defaults to bind)",
			dockerfileContent: `RUN --mount=target=/app echo hello`,
			expectedRules:     []string{},
		},
		{
			name:              "no arguments",
			dockerfileContent: `RUN`,
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

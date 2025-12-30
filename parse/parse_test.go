package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseDockerfile tests the main ParseDockerfile function with various Dockerfile examples.
// It validates that the parser correctly identifies violations based on the content of the Dockerfile.
// Each test case includes:
// - A complete Dockerfile as a string
// - Expected rule codes that should be present in the result
func TestParseDockerfile(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes that should be present
	}{
		{
			name: "FROM with mixed casing - uppercase FROM with lowercase as",
			dockerfileContent: `FROM debian:latest as builder
RUN echo "hello"`,
			expectedRules: []string{"FromAsCasing"},
		},
		{
			name: "FROM with mixed casing - lowercase from with uppercase AS",
			dockerfileContent: `from debian:latest AS builder
RUN echo "hello"`,
			expectedRules: []string{"ConsistentInstructionCasing", "FromAsCasing"},
		},
		{
			name: "FROM with consistent uppercase casing",
			dockerfileContent: `FROM debian:latest AS builder
RUN echo "hello"`,
			expectedRules: []string{},
		},
		{
			name: "FROM with consistent lowercase casing",
			dockerfileContent: `from debian:latest as builder
run echo "hello"`,
			expectedRules: []string{},
		},
		{
			name: "WORKDIR with relative path",
			dockerfileContent: `FROM alpine:latest
WORKDIR usr/share/nginx/html`,
			expectedRules: []string{"WorkdirRelativePath"},
		},
		{
			name: "WORKDIR with absolute path",
			dockerfileContent: `FROM alpine:latest
WORKDIR /usr/share/nginx/html`,
			expectedRules: []string{},
		},
		{
			name: "WORKDIR without argument",
			dockerfileContent: `FROM alpine:latest
WORKDIR`,
			expectedRules: []string{"InvalidInstruction"},
		},
		{
			name: "FROM without argument",
			dockerfileContent: `FROM
RUN echo "hello"`,
			expectedRules: []string{"InvalidInstruction"},
		},
		{
			name: "Multiple violations - FROM casing and WORKDIR relative path",
			dockerfileContent: `FROM debian:latest as builder
WORKDIR app
RUN echo "hello"`,
			expectedRules: []string{"FromAsCasing", "WorkdirRelativePath"},
		},
		{
			name: "Complex Dockerfile with multiple stages - all valid",
			dockerfileContent: `FROM node:18-alpine AS builder
WORKDIR /app
RUN npm install

FROM nginx:alpine AS runtime
WORKDIR /usr/share/nginx/html
RUN echo "done"`,
			expectedRules: []string{},
		},
		{
			name: "Complex Dockerfile with multiple violations",
			dockerfileContent: `FROM node:18-alpine as builder
WORKDIR app
RUN npm install

from nginx:alpine AS runtime
WORKDIR usr/share/nginx/html`,
			expectedRules: []string{"ConsistentInstructionCasing", "FromAsCasing", "WorkdirRelativePath", "FromAsCasing", "WorkdirRelativePath"},
		},
		{
			name: "Simple valid Dockerfile",
			dockerfileContent: `FROM alpine:latest
RUN echo "Hello World"`,
			expectedRules: []string{},
		},
		{
			name:              "Dockerfile with only FROM",
			dockerfileContent: `FROM ubuntu:22.04`,
			expectedRules:     []string{},
		},
		{
			name: "EXPOSE with IP address and port mapping",
			dockerfileContent: `FROM alpine
EXPOSE 127.0.0.1:80:80`,
			expectedRules: []string{"ExposeInvalidFormat"},
		},
		{
			name: "EXPOSE with host-port mapping",
			dockerfileContent: `FROM alpine
EXPOSE 80:80`,
			expectedRules: []string{"ExposeInvalidFormat"},
		},
		{
			name: "EXPOSE with valid port",
			dockerfileContent: `FROM alpine
EXPOSE 80`,
			expectedRules: []string{},
		},
		{
			name: "EXPOSE with valid port and protocol",
			dockerfileContent: `FROM alpine
EXPOSE 80/tcp`,
			expectedRules: []string{},
		},
		{
			name: "EXPOSE without argument",
			dockerfileContent: `FROM alpine
EXPOSE`,
			expectedRules: []string{"InvalidInstruction"},
		},
		{
			name: "EXPOSE with multiple ports - some invalid",
			dockerfileContent: `FROM alpine
EXPOSE 80 8080:8080 443`,
			expectedRules: []string{"ExposeInvalidFormat"},
		},
		{
			name: "WORKDIR with relative path using dot",
			dockerfileContent: `FROM alpine:latest
WORKDIR ./build`,
			expectedRules: []string{"WorkdirRelativePath"},
		},
		{
			name: "WORKDIR with relative path using double dot",
			dockerfileContent: `FROM alpine:latest
WORKDIR ../parent`,
			expectedRules: []string{"WorkdirRelativePath"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)

			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Check the number of rules
			require.Equal(t, len(result.Rules), len(tt.expectedRules), "Number of rules does not match expected")

			// Collect actual rule codes
			actualRuleCodes := make(map[string]bool)
			for _, rule := range result.Rules {
				actualRuleCodes[rule.Code] = true
			}

			// Check that all expected rules are present
			for _, expectedCode := range tt.expectedRules {
				require.True(t, actualRuleCodes[expectedCode],
					"Expected rule %q to be present in results, but it was not found. Actual rules: %v",
					expectedCode, getRuleCodes(result.Rules))
			}
		})
	}
}

func TestParseDockerfile_InvalidSyntax(t *testing.T) {
	t.Run("completely invalid Dockerfile syntax", func(t *testing.T) {
		dockerfileContent := `this is not a valid dockerfile
it has no instructions
just random text`

		result, err := ParseDockerfile(dockerfileContent)

		// The parser might not return an error for unknown instructions
		// but should still return a result
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestParseDockerfile_EmptyDockerfile(t *testing.T) {
	t.Run("empty Dockerfile", func(t *testing.T) {
		result, err := ParseDockerfile("")

		// Empty Dockerfiles return an error from the parser
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "no instructions")
	})
}

func TestParseDockerfile_RuleDetails(t *testing.T) {
	t.Run("verify rule details for FromAsCasing", func(t *testing.T) {
		dockerfileContent := `FROM debian:latest as builder`

		result, err := ParseDockerfile(dockerfileContent)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Rules, 1, "Expected exactly 1 rule")

		rule := result.Rules[0]
		require.Equal(t, "FromAsCasing", rule.Code)
		require.NotEmpty(t, rule.Description, "Rule should have a description")
		require.NotEmpty(t, rule.Url, "Rule should have a documentation URL")
		require.Equal(t, 1, rule.StartLine, "Rule should start at line 1")
		require.Equal(t, 1, rule.EndLine, "Rule should end at line 1")
	})

	t.Run("verify rule details for WorkdirRelativePath", func(t *testing.T) {
		dockerfileContent := `FROM alpine:latest
WORKDIR app`

		result, err := ParseDockerfile(dockerfileContent)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Rules, 1, "Expected exactly 1 rule")

		rule := result.Rules[0]
		require.Equal(t, "WorkdirRelativePath", rule.Code)
		require.NotEmpty(t, rule.Description, "Rule should have a description")
		require.NotEmpty(t, rule.Url, "Rule should have a documentation URL")
		require.Equal(t, 2, rule.StartLine, "Rule should start at line 2")
		require.Equal(t, 2, rule.EndLine, "Rule should end at line 2")
	})

	t.Run("verify rule details for InvalidInstruction", func(t *testing.T) {
		dockerfileContent := `FROM alpine:latest
WORKDIR`

		result, err := ParseDockerfile(dockerfileContent)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Rules, 1, "Expected exactly 1 rule")

		rule := result.Rules[0]
		require.Equal(t, "InvalidInstruction", rule.Code)
		require.NotEmpty(t, rule.Description, "Rule should have a description")
		require.Equal(t, 2, rule.StartLine, "Rule should start at line 2")
		require.Equal(t, 2, rule.EndLine, "Rule should end at line 2")
	})
}

func TestParseDockerfile_MultipleViolationsSameLine(t *testing.T) {
	t.Run("single line can only have one violation per instruction", func(t *testing.T) {
		// Each instruction is on its own line, so we test multiple instructions
		dockerfileContent := `FROM debian:latest as builder
WORKDIR app`

		result, err := ParseDockerfile(dockerfileContent)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Rules, 2, "Expected 2 rules (one per violation)")

		// Verify we have both violations
		ruleCodes := getRuleCodes(result.Rules)
		require.Contains(t, ruleCodes, "FromAsCasing")
		require.Contains(t, ruleCodes, "WorkdirRelativePath")
	})
}

func TestParseDockerfile_CompleteValidDockerfile(t *testing.T) {
	t.Run("comprehensive valid Dockerfile with multiple instructions", func(t *testing.T) {
		dockerfileContent := `# A comment
# FROM instruction valid examples

ARG NODE_VERSION=18
FROM debian:latest
FROM alpine:3.18
FROM debian:latest AS builder
FROM debian:latest AS deb-builder
FROM node:${NODE_VERSION}-alpine AS node-builder

# RUN instruction valid examples
RUN npm ci --only=production
RUN npm run build

# CMD instruction valid examples
CMD ["nginx", "-g", "daemon off;"]

# LABEL instruction valid examples
LABEL maintainer="dev@example.com"
LABEL version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/example/repo"

# EXPOSE instruction valid examples
EXPOSE 80
EXPOSE 443

# ENV instruction valid examples
ENV NODE_ENV=production
ENV APP_PORT=3000

# ADD instruction valid examples
ADD file1.txt file2.txt /usr/src/things/
ADD https://example.com/archive.zip /usr/src/things/
ADD git@github.com:user/repo.git /usr/src/things/

# COPY instruction valid examples
COPY package*.json ./
COPY . .
COPY --from=builder /app/dist .

# ENTRYPOINT instruction valid examples
ENTRYPOINT ["executable", "param1", "param2"]

# VOLUME instruction valid examples
VOLUME ["/var/log/nginx"]

# USER instruction valid examples
USER nginx

# WORKDIR instruction valid examples
WORKDIR /app
WORKDIR /usr/share/nginx/html

# ARG instruction valid examples

ARG NODE_VERSION=18
ARG APP_ENV=production

# ONBUILD instruction valid examples
ONBUILD RUN echo "building"
ONBUILD ADD . /app
ONBUILD COPY requirements.txt /app/

# STOPSIGNAL instruction valid examples
STOPSIGNAL SIGTERM

# HEALTHCHECK instruction valid examples
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget --quiet --tries=1 --spider http://localhost/ || exit 1

# SHELL instruction valid examples
SHELL ["cmd", "/S", "/C"]
`

		result, err := ParseDockerfile(dockerfileContent)

		require.NoError(t, err, "ParseDockerfile should not return an error for valid Dockerfile")
		require.NotNil(t, result, "ParseDockerfile should return a non-nil result")
		require.Empty(t, result.Rules, "Expected no rules for valid Dockerfile, but got: %v", getRuleCodes(result.Rules))
	})
}

func TestParseDockerfile_CompleteInvalidDockerfile(t *testing.T) {
	t.Run("comprehensive invalid Dockerfile with violations for each instruction", func(t *testing.T) {
		dockerfileContent := `# Invalid Dockerfile with one violation per instruction type
# ARG instruction invalid example
ARG 1INVALID=value

# FROM instruction invalid example
FROM debian:latest as builder

# RUN instruction invalid example
RUN --network=invalid echo "test"

# CMD instruction invalid example
CMD ['single', 'quotes']

# LABEL instruction invalid example
LABEL version 1.0

# EXPOSE instruction invalid example
EXPOSE 80:8080

# ENV instruction invalid example
ENV =invalid

# ADD instruction invalid example
ADD --from=invalid file.txt /dest/

# COPY instruction invalid example
COPY --invalid file.txt /dest/

# ENTRYPOINT instruction invalid example
ENTRYPOINT ['single', 'quotes']

# VOLUME instruction invalid example
VOLUME ['/data']

# USER instruction invalid example
USER user:group:extra

# WORKDIR instruction invalid example
WORKDIR relative/path

# ONBUILD instruction invalid example
ONBUILD

# STOPSIGNAL instruction invalid example
STOPSIGNAL

# HEALTHCHECK instruction invalid example
HEALTHCHECK

# SHELL instruction invalid example
SHELL ['/bin/sh', '-c']
`

		result, err := ParseDockerfile(dockerfileContent)

		require.NoError(t, err, "ParseDockerfile should not return an error even with invalid instructions")
		require.NotNil(t, result, "ParseDockerfile should return a non-nil result")
		require.NotEmpty(t, result.Rules, "Expected rules for invalid Dockerfile")

		// Collect actual rule codes
		actualRuleCodes := getRuleCodes(result.Rules)

		// Expected violations - one per instruction type
		expectedRules := []string{
			"ArgInvalidFormat",          // ARG 1INVALID=value
			"FromAsCasing",              // FROM debian:latest as builder
			"RunInvalidNetworkFlag",     // RUN --network=invalid
			"CmdInvalidExecForm",        // CMD ['single', 'quotes']
			"LabelInvalidFormat",        // LABEL version 1.0
			"ExposeInvalidFormat",       // EXPOSE 80:8080
			"EnvInvalidFormat",          // ENV =invalid
			"AddInvalidFlag",            // ADD --from=invalid
			"CopyInvalidFlag",           // COPY --invalid
			"EntrypointInvalidExecForm", // ENTRYPOINT ['single', 'quotes']
			"VolumeInvalidJsonForm",     // VOLUME ['/data']
			"UserInvalidFormat",         // USER user:group:extra
			"WorkdirRelativePath",       // WORKDIR relative/path
			"InvalidInstruction",        // ONBUILD, STOPSIGNAL, HEALTHCHECK (3 instances)
			"ShellInvalidJsonForm",      // SHELL ['/bin/sh', '-c']
		}

		// Create a map of actual rule codes for easy lookup
		actualRuleMap := make(map[string]bool)
		for _, code := range actualRuleCodes {
			actualRuleMap[code] = true
		}

		// Verify each expected rule is present
		var missingRules []string
		for _, expectedCode := range expectedRules {
			if !actualRuleMap[expectedCode] {
				missingRules = append(missingRules, expectedCode)
			}
		}

		require.Empty(t, missingRules, "Missing expected rule codes: %v. Got rules: %v", missingRules, actualRuleCodes)

		t.Logf("Found %d rules as expected: %v", len(result.Rules), actualRuleCodes)
	})
}

// Helper function to extract rule codes from a slice of Rules
func getRuleCodes(rules []Rule) []string {
	codes := make([]string, 0, len(rules))
	for _, rule := range rules {
		codes = append(codes, rule.Code)
	}
	return codes
}

package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckSecretsInArgOrEnv(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		// Valid cases - no secrets
		{
			name: "normal ENV variable",
			dockerfileContent: `FROM alpine
ENV NODE_ENV=production`,
			expectViolation: false,
		},
		{
			name: "normal ARG variable",
			dockerfileContent: `FROM alpine
ARG VERSION=1.0`,
			expectViolation: false,
		},
		{
			name: "multiple normal variables",
			dockerfileContent: `FROM alpine
ENV APP_NAME=myapp
ENV PORT=8080
ARG BUILD_VERSION=1.0`,
			expectViolation: false,
		},
		// Invalid cases - secrets in ENV
		{
			name: "ENV with SECRET in name",
			dockerfileContent: `FROM alpine
ENV MY_SECRET=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENV with PASSWORD in name",
			dockerfileContent: `FROM alpine
ENV DB_PASSWORD=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENV with TOKEN in name",
			dockerfileContent: `FROM alpine
ENV AUTH_TOKEN=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENV with API_KEY in name",
			dockerfileContent: `FROM alpine
ENV API_KEY=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENV with APIKEY in name",
			dockerfileContent: `FROM alpine
ENV APIKEY=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENV with PRIVATE_KEY in name",
			dockerfileContent: `FROM alpine
ENV PRIVATE_KEY=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENV AWS_SECRET_ACCESS_KEY",
			dockerfileContent: `FROM alpine
ENV AWS_SECRET_ACCESS_KEY=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ENV AWS_ACCESS_KEY_ID",
			dockerfileContent: `FROM alpine
ENV AWS_ACCESS_KEY_ID=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		// Invalid cases - secrets in ARG
		{
			name: "ARG with SECRET in name",
			dockerfileContent: `FROM alpine
ARG MY_SECRET`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ARG with PASSWORD in name",
			dockerfileContent: `FROM alpine
ARG DB_PASSWORD=default`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ARG with TOKEN in name",
			dockerfileContent: `FROM alpine
ARG GITHUB_TOKEN`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "ARG AWS_SECRET_ACCESS_KEY",
			dockerfileContent: `FROM alpine
ARG AWS_SECRET_ACCESS_KEY`,
			expectViolation: true,
			expectedCount:   1,
		},
		// Multiple violations
		{
			name: "multiple ENV with secrets",
			dockerfileContent: `FROM alpine
ENV DB_PASSWORD=value
ENV API_KEY=value`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "multiple ARG with secrets",
			dockerfileContent: `FROM alpine
ARG DB_PASSWORD
ARG API_TOKEN`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "mixed ENV and ARG with secrets",
			dockerfileContent: `FROM alpine
ARG DB_PASSWORD
ENV API_KEY=value`,
			expectViolation: true,
			expectedCount:   2,
		},
		// Case insensitivity
		{
			name: "lowercase secret",
			dockerfileContent: `FROM alpine
ENV my_secret=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "mixed case password",
			dockerfileContent: `FROM alpine
ENV MyPassword=value`,
			expectViolation: false, // camelCase without word boundaries is not detected
		},
		// Common variations
		{
			name: "passwd variation",
			dockerfileContent: `FROM alpine
ENV DB_PASSWD=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "pwd variation",
			dockerfileContent: `FROM alpine
ENV DB_PWD=value`,
			expectViolation: false, // "pwd" is not in the moby token list
		},
		{
			name: "auth in name",
			dockerfileContent: `FROM alpine
ENV AUTH_STRING=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "credential in name",
			dockerfileContent: `FROM alpine
ENV CREDENTIALS=value`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "cred in name",
			dockerfileContent: `FROM alpine
ENV DB_CREDS=value`,
			expectViolation: false, // "cred" short form is not in the moby token list (but "credential" and "credentials" are)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for SecretsUsedInArgOrEnv rules
			secretRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "SecretsUsedInArgOrEnv" {
					secretRules = append(secretRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, secretRules,
					"Expected no SecretsUsedInArgOrEnv violations but got: %v", secretRules)
			} else {
				require.Len(t, secretRules, tt.expectedCount,
					"Expected %d SecretsUsedInArgOrEnv violations but got %d: %v",
					tt.expectedCount, len(secretRules), secretRules)

				// Verify rule structure
				for _, rule := range secretRules {
					require.Equal(t, "SecretsUsedInArgOrEnv", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "Sensitive data")
					require.Contains(t, rule.Description, "secret mounts")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/secrets-used-in-arg-or-env/",
						rule.Url)
				}
			}
		})
	}
}

func TestCheckSecretsInArgOrEnvComplexScenarios(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedCount     int
	}{
		{
			name: "mixed secrets and normal variables",
			dockerfileContent: `FROM alpine
ENV NODE_ENV=production
ENV API_KEY=value
ENV PORT=8080
ARG VERSION=1.0
ARG DB_PASSWORD`,
			expectedCount: 2, // API_KEY and DB_PASSWORD
		},
		{
			name: "multi-stage with secrets in different stages",
			dockerfileContent: `FROM alpine AS builder
ARG BUILD_TOKEN

FROM alpine AS runtime
ENV API_SECRET=value`,
			expectedCount: 2, // Both stages have secrets
		},
		{
			name: "multiple secrets in single ENV line",
			dockerfileContent: `FROM alpine
ENV SECRET_KEY=value API_TOKEN=value`,
			expectedCount: 2, // Both SECRET_KEY and API_TOKEN
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for SecretsUsedInArgOrEnv rules
			secretRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "SecretsUsedInArgOrEnv" {
					secretRules = append(secretRules, rule)
				}
			}

			require.Len(t, secretRules, tt.expectedCount,
				"Expected %d SecretsUsedInArgOrEnv violations but got %d: %v",
				tt.expectedCount, len(secretRules), secretRules)
		})
	}
}

func TestIsSensitiveVariableName(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		// Sensitive names - at word boundaries
		{"secret at end", "MY_SECRET", true},
		{"password at end", "DB_PASSWORD", true},
		{"token at end", "AUTH_TOKEN", true},
		{"key at end", "API_KEY", true},
		{"apikey standalone", "APIKEY", true},
		{"key at start", "KEY_NAME", true},
		{"secret at start", "SECRET_VALUE", true},
		{"passwd at end", "DB_PASSWD", true},
		{"pword at end", "DB_PWORD", true},
		{"auth standalone", "AUTH", true},
		{"credentials at end", "DB_CREDENTIALS", true},
		{"credential at end", "AWS_CREDENTIAL", true},
		{"token standalone", "TOKEN", true},

		// Case insensitive
		{"lowercase secret", "my_secret", true},
		{"lowercase password", "password", true},
		{"uppercase token", "AUTH_TOKEN", true},

		// With allowlist - should NOT be flagged
		{"public key", "PUBLIC_KEY", false},
		{"public token", "PUBLIC_TOKEN", false},

		// Not at word boundaries - should NOT be flagged with new regex
		{"password in middle", "MYPASSWORDTEST", false},
		{"secret in middle", "MYSECRETTEST", false},
		{"camelCase password", "MyPassword", false},
		{"camelCase secret", "MySecret", false},

		// Non-sensitive names
		{"normal var", "NODE_ENV", false},
		{"port", "PORT", false},
		{"version", "VERSION", false},
		{"path", "APP_PATH", false},
		{"url", "DATABASE_URL", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveVariableName(tt.varName)
			require.Equal(t, tt.expected, result,
				"isSensitiveVariableName(%q) returned unexpected result", tt.varName)
		})
	}
}

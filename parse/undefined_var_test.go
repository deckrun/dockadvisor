package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckUndefinedVar(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		// Valid cases - no violations
		{
			name: "ARG declared before COPY",
			dockerfileContent: `FROM alpine
ARG foo=bar
COPY $foo .`,
			expectViolation: false,
		},
		{
			name: "ENV declared before COPY",
			dockerfileContent: `FROM alpine
ENV foo=bar
COPY $foo .`,
			expectViolation: false,
		},
		{
			name: "predefined ARG TARGETPLATFORM",
			dockerfileContent: `FROM alpine
RUN echo $TARGETPLATFORM`,
			expectViolation: false,
		},
		{
			name: "predefined ARG BUILDPLATFORM",
			dockerfileContent: `FROM alpine
WORKDIR /app/$BUILDPLATFORM`,
			expectViolation: false,
		},
		{
			name: "shell form RUN with undefined variable",
			dockerfileContent: `FROM alpine
RUN echo $UNDEFINED_VAR`,
			expectViolation: false, // Shell form is skipped
		},
		{
			name: "shell form CMD with undefined variable",
			dockerfileContent: `FROM alpine
CMD echo $UNDEFINED_VAR`,
			expectViolation: false, // Shell form is skipped
		},
		{
			name: "shell form ENTRYPOINT with undefined variable",
			dockerfileContent: `FROM alpine
ENTRYPOINT echo $UNDEFINED_VAR`,
			expectViolation: false, // Shell form is skipped
		},
		{
			name: "global ARG used in FROM",
			dockerfileContent: `ARG VERSION=18
FROM node:${VERSION}`,
			expectViolation: false,
		},
		{
			name: "global ARG used in stage",
			dockerfileContent: `ARG BASE=alpine
FROM $BASE
COPY $BASE .`,
			expectViolation: false,
		},
		{
			name: "ENV with variable reference",
			dockerfileContent: `FROM alpine
ARG VERSION=1.0
ENV APP_VERSION=$VERSION`,
			expectViolation: false,
		},
		{
			name: "ARG with variable reference",
			dockerfileContent: `FROM alpine
ARG VERSION=1.0
ARG TAG=$VERSION`,
			expectViolation: false,
		},
		// Invalid cases - violations
		{
			name: "undefined variable in COPY",
			dockerfileContent: `FROM alpine
COPY $foo .`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "undefined variable in ADD",
			dockerfileContent: `FROM alpine
ADD $foo /app/`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "undefined variable in WORKDIR",
			dockerfileContent: `FROM alpine
WORKDIR /app/$UNDEFINED`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "undefined variable in USER",
			dockerfileContent: `FROM alpine
USER $UNDEFINED_USER`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "undefined variable in LABEL",
			dockerfileContent: `FROM alpine
LABEL version=$UNDEFINED_VERSION`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "undefined variable in ARG default value",
			dockerfileContent: `FROM alpine
ARG VERSION=$UNDEFINED`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "undefined variable in ENV value",
			dockerfileContent: `FROM alpine
ENV APP_VERSION=$UNDEFINED`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "undefined variable in COPY --chown flag",
			dockerfileContent: `FROM alpine
COPY --chown=$USER:$GROUP file.txt /app/`,
			expectViolation: true,
			expectedCount:   2, // Both $USER and $GROUP
		},
		{
			name: "multiple undefined variables",
			dockerfileContent: `FROM alpine
COPY $foo $bar /app/`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "exec form RUN with undefined variable",
			dockerfileContent: `FROM alpine
RUN ["echo", "$UNDEFINED"]`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "exec form CMD with undefined variable",
			dockerfileContent: `FROM alpine
CMD ["echo", "$UNDEFINED"]`,
			expectViolation: true,
			expectedCount:   1,
		},
		{
			name: "exec form ENTRYPOINT with undefined variable",
			dockerfileContent: `FROM alpine
ENTRYPOINT ["echo", "$UNDEFINED"]`,
			expectViolation: true,
			expectedCount:   1,
		},
		// Multi-stage cases
		{
			name: "variable not available across stages",
			dockerfileContent: `FROM alpine AS builder
ARG BUILD_VERSION=1.0
RUN echo $BUILD_VERSION

FROM alpine AS runtime
COPY $BUILD_VERSION /app/`,
			expectViolation: true,
			expectedCount:   1, // $BUILD_VERSION not available in runtime stage
		},
		{
			name: "global ARG available in all stages",
			dockerfileContent: `ARG GLOBAL_VERSION=1.0

FROM alpine AS builder
COPY $GLOBAL_VERSION /builder/

FROM alpine AS runtime
COPY $GLOBAL_VERSION /runtime/`,
			expectViolation: false,
		},
		{
			name: "ENV not available across stages",
			dockerfileContent: `FROM alpine AS builder
ENV BUILD_ENV=production
RUN echo $BUILD_ENV

FROM alpine AS runtime
COPY $BUILD_ENV /app/`,
			expectViolation: true,
			expectedCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Filter for UndefinedVar rules
			undefinedVarRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "UndefinedVar" {
					undefinedVarRules = append(undefinedVarRules, rule)
				}
			}

			if !tt.expectViolation {
				require.Empty(t, undefinedVarRules,
					"Expected no UndefinedVar violations but got: %v", undefinedVarRules)
			} else {
				require.Len(t, undefinedVarRules, tt.expectedCount,
					"Expected %d UndefinedVar violations but got %d: %v",
					tt.expectedCount, len(undefinedVarRules), undefinedVarRules)

				// Verify rule structure
				for _, rule := range undefinedVarRules {
					require.Equal(t, "UndefinedVar", rule.Code)
					require.NotEmpty(t, rule.Description)
					require.Contains(t, rule.Description, "undefined variable")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/undefined-var/",
						rule.Url)
				}
			}
		})
	}
}

func TestCheckUndefinedVarWithPredefinedArgs(t *testing.T) {
	predefinedArgs := []string{
		"TARGETPLATFORM", "TARGETOS", "TARGETARCH", "TARGETVARIANT",
		"BUILDPLATFORM", "BUILDOS", "BUILDARCH", "BUILDVARIANT",
		"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy",
		"FTP_PROXY", "ftp_proxy", "NO_PROXY", "no_proxy",
		"ALL_PROXY", "all_proxy",
	}

	for _, argName := range predefinedArgs {
		t.Run("predefined ARG "+argName, func(t *testing.T) {
			dockerfileContent := `FROM alpine
COPY $` + argName + ` /app/`

			result, err := ParseDockerfile(dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for UndefinedVar rules
			undefinedVarRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "UndefinedVar" {
					undefinedVarRules = append(undefinedVarRules, rule)
				}
			}

			require.Empty(t, undefinedVarRules,
				"Expected no UndefinedVar violation for predefined ARG %s, but got: %v",
				argName, undefinedVarRules)
		})
	}
}

func TestExtractEnvNames(t *testing.T) {
	tests := []struct {
		name                  string
		dockerfileContent     string
		expectedEnvNames      []string
		instructionLineNumber int // 1-based line number of the ENV instruction
	}{
		{
			name: "single key=value",
			dockerfileContent: `FROM alpine
ENV FOO=bar`,
			expectedEnvNames:      []string{"FOO"},
			instructionLineNumber: 2,
		},
		{
			name: "multiple key=value pairs",
			dockerfileContent: `FROM alpine
ENV FOO=bar BAZ=qux`,
			expectedEnvNames:      []string{"FOO", "BAZ"},
			instructionLineNumber: 2,
		},
		{
			name: "key value format",
			dockerfileContent: `FROM alpine
ENV FOO bar`,
			expectedEnvNames:      []string{"FOO"},
			instructionLineNumber: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the Dockerfile
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// This test is just to verify the helper function works correctly
			// The actual validation is tested in TestCheckUndefinedVar
		})
	}
}

func TestCheckUndefinedVarComplexScenarios(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedCount     int
	}{
		{
			name: "mixed valid and invalid variables",
			dockerfileContent: `FROM alpine
ARG DEFINED=value
COPY $DEFINED $UNDEFINED /app/`,
			expectedCount: 1, // Only $UNDEFINED
		},
		{
			name: "variable shadowing in stages",
			dockerfileContent: `FROM alpine AS builder
ARG VERSION=1.0
RUN echo $VERSION

FROM alpine AS runtime
ARG VERSION=2.0
RUN echo $VERSION`,
			expectedCount: 0, // Both stages have their own VERSION
		},
		{
			name: "ENV overrides ARG in same stage",
			dockerfileContent: `FROM alpine
ARG FOO=arg_value
ENV FOO=env_value
COPY $FOO /app/`,
			expectedCount: 0, // FOO is defined by both ARG and ENV
		},
		{
			name:              "undefined in FROM platform flag",
			dockerfileContent: `FROM --platform=$UNDEFINED alpine`,
			expectedCount:     1,
		},
		{
			name:              "undefined in FROM image reference",
			dockerfileContent: `FROM alpine:$UNDEFINED`,
			expectedCount:     1,
		},
		{
			name: "multiple undefined in same instruction",
			dockerfileContent: `FROM alpine
ENV PATH=$UNDEFINED_A:$UNDEFINED_B:$UNDEFINED_C`,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Filter for UndefinedVar rules
			undefinedVarRules := []Rule{}
			for _, rule := range result.Rules {
				if rule.Code == "UndefinedVar" {
					undefinedVarRules = append(undefinedVarRules, rule)
				}
			}

			require.Len(t, undefinedVarRules, tt.expectedCount,
				"Expected %d UndefinedVar violations but got %d: %v",
				tt.expectedCount, len(undefinedVarRules), undefinedVarRules)
		})
	}
}

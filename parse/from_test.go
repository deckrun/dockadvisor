package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckFromAsCasing(t *testing.T) {
	tests := []struct {
		name         string
		fromKeyword  string
		originalLine string
		expected     bool
	}{
		// Good examples - consistent casing
		{
			name:         "uppercase FROM and AS",
			fromKeyword:  "FROM",
			originalLine: "FROM debian:latest AS builder",
			expected:     true,
		},
		{
			name:         "uppercase FROM and AS with different stage name",
			fromKeyword:  "FROM",
			originalLine: "FROM debian:latest AS deb-builder",
			expected:     true,
		},
		{
			name:         "lowercase from and as",
			fromKeyword:  "from",
			originalLine: "from debian:latest as deb-builder",
			expected:     true,
		},
		{
			name:         "lowercase from and as with simple stage name",
			fromKeyword:  "from",
			originalLine: "from debian:latest as builder",
			expected:     true,
		},
		{
			name:         "no AS keyword - should be valid",
			fromKeyword:  "FROM",
			originalLine: "FROM debian:latest",
			expected:     true,
		},
		{
			name:         "uppercase FROM without AS",
			fromKeyword:  "FROM",
			originalLine: "FROM ubuntu:22.04",
			expected:     true,
		},
		{
			name:         "lowercase from without as",
			fromKeyword:  "from",
			originalLine: "from alpine:3.18",
			expected:     true,
		},
		// Bad examples - mixed casing
		{
			name:         "uppercase FROM with lowercase as",
			fromKeyword:  "FROM",
			originalLine: "FROM debian:latest as builder",
			expected:     false,
		},
		{
			name:         "lowercase from with uppercase AS",
			fromKeyword:  "from",
			originalLine: "from debian:latest AS builder",
			expected:     false,
		},
		{
			name:         "uppercase FROM with lowercase as and complex stage name",
			fromKeyword:  "FROM",
			originalLine: "FROM node:18-alpine as node-builder",
			expected:     false,
		},
		{
			name:         "lowercase from with uppercase AS and complex stage name",
			fromKeyword:  "from",
			originalLine: "from python:3.11 AS python-builder",
			expected:     false,
		},
		// Edge cases
		{
			name:         "mixed case As (title case) with uppercase FROM - should be invalid",
			fromKeyword:  "FROM",
			originalLine: "FROM debian:latest As builder",
			expected:     false,
		},
		{
			name:         "mixed case aS with uppercase FROM - should be invalid",
			fromKeyword:  "FROM",
			originalLine: "FROM debian:latest aS builder",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkFromAsCasing(tt.fromKeyword, tt.originalLine)
			require.Equal(t, tt.expected, result, "checkFromAsCasing(%q, %q) returned unexpected result", tt.fromKeyword, tt.originalLine)
		})
	}
}

func TestCheckImageReferenceFormat(t *testing.T) {
	tests := []struct {
		name     string
		imageRef string
		expected bool
	}{
		// Valid image references
		{
			name:     "simple image name",
			imageRef: "debian",
			expected: true,
		},
		{
			name:     "image with tag",
			imageRef: "debian:latest",
			expected: true,
		},
		{
			name:     "image with version tag",
			imageRef: "ubuntu:22.04",
			expected: true,
		},
		{
			name:     "image with digest",
			imageRef: "alpine@sha256:abc123def456",
			expected: true,
		},
		{
			name:     "image with registry",
			imageRef: "docker.io/library/debian",
			expected: true,
		},
		{
			name:     "image with registry and tag",
			imageRef: "gcr.io/project/image:v1.0.0",
			expected: true,
		},
		{
			name:     "image with registry, port, and tag",
			imageRef: "localhost:5000/myimage:latest",
			expected: true,
		},
		{
			name:     "image with variable",
			imageRef: "base:${CODE_VERSION}",
			expected: true,
		},
		{
			name:     "image with dollar variable",
			imageRef: "busybox:$VERSION",
			expected: true,
		},
		{
			name:     "image with variable default value",
			imageRef: "alpine:${TAG:-3.14}",
			expected: true,
		},
		{
			name:     "image with variable default value (complex)",
			imageRef: "ubuntu:${VERSION:-22.04-jammy}",
			expected: true,
		},
		{
			name:     "complete image with variable default",
			imageRef: "registry.example.com/app:${TAG:-latest}",
			expected: true,
		},
		{
			name:     "image name as variable with default",
			imageRef: "${IMAGE:-alpine}:3.14",
			expected: true,
		},
		{
			name:     "complex registry with path",
			imageRef: "registry.example.com:8080/team/project/image:tag",
			expected: true,
		},
		// Invalid image references
		{
			name:     "empty string",
			imageRef: "",
			expected: false,
		},
		{
			name:     "whitespace only",
			imageRef: "   ",
			expected: false,
		},
		{
			name:     "image with spaces",
			imageRef: "my image:latest",
			expected: false,
		},
		{
			name:     "image with multiple @",
			imageRef: "image@sha256:abc@def",
			expected: false,
		},
		{
			name:     "image with invalid digest format",
			imageRef: "image@invaliddigest",
			expected: false,
		},
		{
			name:     "image with special characters",
			imageRef: "image!@#$:latest",
			expected: false,
		},
		{
			name:     "unsupported :+ expansion",
			imageRef: "alpine:${TAG:+value}",
			expected: false,
		},
		{
			name:     "unsupported :? expansion",
			imageRef: "alpine:${TAG:?error}",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkImageReferenceFormat(tt.imageRef)
			require.Equal(t, tt.expected, result, "checkImageReferenceFormat(%q) returned unexpected result", tt.imageRef)
		})
	}
}

func TestCheckRedundantTargetPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected bool
	}{
		// Redundant cases
		{
			name:     "$TARGETPLATFORM is redundant",
			platform: "$TARGETPLATFORM",
			expected: true,
		},
		{
			name:     "$TARGETPLATFORM with spaces is redundant",
			platform: "  $TARGETPLATFORM  ",
			expected: true,
		},
		// Non-redundant cases
		{
			name:     "linux/amd64 is not redundant",
			platform: "linux/amd64",
			expected: false,
		},
		{
			name:     "linux/arm64 is not redundant",
			platform: "linux/arm64",
			expected: false,
		},
		{
			name:     "$BUILDPLATFORM is not redundant",
			platform: "$BUILDPLATFORM",
			expected: false,
		},
		{
			name:     "windows/amd64 is not redundant",
			platform: "windows/amd64",
			expected: false,
		},
		{
			name:     "empty string is not redundant",
			platform: "",
			expected: false,
		},
		{
			name:     "lowercase targetplatform is not redundant",
			platform: "$targetplatform",
			expected: false,
		},
		{
			name:     "TARGETPLATFORM without $ is not redundant",
			platform: "TARGETPLATFORM",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkRedundantTargetPlatform(tt.platform)
			require.Equal(t, tt.expected, result, "checkRedundantTargetPlatform(%q) returned unexpected result", tt.platform)
		})
	}
}

func TestCheckPlatformFormat(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected bool
	}{
		// Valid platform formats
		{
			name:     "linux/amd64",
			platform: "linux/amd64",
			expected: true,
		},
		{
			name:     "linux/arm64",
			platform: "linux/arm64",
			expected: true,
		},
		{
			name:     "windows/amd64",
			platform: "windows/amd64",
			expected: true,
		},
		{
			name:     "linux only",
			platform: "linux",
			expected: true,
		},
		{
			name:     "linux/arm64/v8",
			platform: "linux/arm64",
			expected: true,
		},
		{
			name:     "variable platform",
			platform: "$BUILDPLATFORM",
			expected: true,
		},
		{
			name:     "linux/386",
			platform: "linux/386",
			expected: true,
		},
		// Invalid platform formats
		{
			name:     "empty string",
			platform: "",
			expected: false,
		},
		{
			name:     "whitespace only",
			platform: "   ",
			expected: false,
		},
		{
			name:     "invalid OS",
			platform: "invalid/amd64",
			expected: false,
		},
		{
			name:     "invalid architecture",
			platform: "linux/invalid",
			expected: false,
		},
		{
			name:     "too many parts",
			platform: "linux/amd64/variant/extra",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkPlatformFormat(tt.platform)
			require.Equal(t, tt.expected, result, "checkPlatformFormat(%q) returned unexpected result", tt.platform)
		})
	}
}

func TestCheckPlatformConstant(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected bool // true if constant (violation), false if variable (ok)
	}{
		// Constants (violations)
		{
			name:     "linux/amd64 constant",
			platform: "linux/amd64",
			expected: true,
		},
		{
			name:     "linux/arm64 constant",
			platform: "linux/arm64",
			expected: true,
		},
		{
			name:     "windows/amd64 constant",
			platform: "windows/amd64",
			expected: true,
		},
		{
			name:     "linux constant",
			platform: "linux",
			expected: true,
		},
		// Variables (allowed)
		{
			name:     "$BUILDPLATFORM variable",
			platform: "$BUILDPLATFORM",
			expected: false,
		},
		{
			name:     "$TARGETPLATFORM variable",
			platform: "$TARGETPLATFORM",
			expected: false,
		},
		{
			name:     "${BUILDPLATFORM} variable",
			platform: "${BUILDPLATFORM}",
			expected: false,
		},
		{
			name:     "${TARGETPLATFORM} variable",
			platform: "${TARGETPLATFORM}",
			expected: false,
		},
		{
			name:     "$CUSTOM_PLATFORM variable",
			platform: "$CUSTOM_PLATFORM",
			expected: false,
		},
		// Edge cases
		{
			name:     "empty string",
			platform: "",
			expected: false,
		},
		{
			name:     "whitespace",
			platform: "   ",
			expected: false, // Whitespace trims to empty, which returns false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkPlatformConstant(tt.platform)
			require.Equal(t, tt.expected, result,
				"checkPlatformConstant(%q) returned unexpected result", tt.platform)
		})
	}
}

func TestCheckStageNameFormat(t *testing.T) {
	tests := []struct {
		name      string
		stageName string
		expected  bool
	}{
		// Valid stage names
		{
			name:      "simple name",
			stageName: "builder",
			expected:  true,
		},
		{
			name:      "name with dash",
			stageName: "deb-builder",
			expected:  true,
		},
		{
			name:      "name with underscore",
			stageName: "my_builder",
			expected:  true,
		},
		{
			name:      "name starting with underscore",
			stageName: "_builder",
			expected:  true,
		},
		{
			name:      "name with dot",
			stageName: "builder.v1",
			expected:  true,
		},
		{
			name:      "complex name",
			stageName: "my_build-stage.v1",
			expected:  true,
		},
		{
			name:      "uppercase name",
			stageName: "BUILDER",
			expected:  true,
		},
		{
			name:      "mixed case name",
			stageName: "MyBuilder",
			expected:  true,
		},
		// Invalid stage names
		{
			name:      "empty string",
			stageName: "",
			expected:  false,
		},
		{
			name:      "whitespace only",
			stageName: "   ",
			expected:  false,
		},
		{
			name:      "starting with digit",
			stageName: "1builder",
			expected:  false,
		},
		{
			name:      "starting with dash",
			stageName: "-builder",
			expected:  false,
		},
		{
			name:      "with spaces",
			stageName: "my builder",
			expected:  false,
		},
		{
			name:      "with special characters",
			stageName: "builder@v1",
			expected:  false,
		},
		{
			name:      "with exclamation",
			stageName: "builder!",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkStageNameFormat(tt.stageName)
			require.Equal(t, tt.expected, result, "checkStageNameFormat(%q) returned unexpected result", tt.stageName)
		})
	}
}

func TestCheckReservedStageName(t *testing.T) {
	tests := []struct {
		name      string
		stageName string
		expected  bool
	}{
		// Reserved stage names
		{
			name:      "lowercase 'scratch' is reserved",
			stageName: "scratch",
			expected:  true,
		},
		{
			name:      "uppercase 'SCRATCH' is reserved",
			stageName: "SCRATCH",
			expected:  true,
		},
		{
			name:      "mixed case 'Scratch' is reserved",
			stageName: "Scratch",
			expected:  true,
		},
		{
			name:      "lowercase 'context' is reserved",
			stageName: "context",
			expected:  true,
		},
		{
			name:      "uppercase 'CONTEXT' is reserved",
			stageName: "CONTEXT",
			expected:  true,
		},
		{
			name:      "mixed case 'Context' is reserved",
			stageName: "Context",
			expected:  true,
		},
		{
			name:      "mixed case 'CoNtExT' is reserved",
			stageName: "CoNtExT",
			expected:  true,
		},
		// Non-reserved stage names
		{
			name:      "builder is not reserved",
			stageName: "builder",
			expected:  false,
		},
		{
			name:      "base is not reserved",
			stageName: "base",
			expected:  false,
		},
		{
			name:      "scratch-builder is not reserved",
			stageName: "scratch-builder",
			expected:  false,
		},
		{
			name:      "context-builder is not reserved",
			stageName: "context-builder",
			expected:  false,
		},
		{
			name:      "my-scratch is not reserved",
			stageName: "my-scratch",
			expected:  false,
		},
		{
			name:      "my-context is not reserved",
			stageName: "my-context",
			expected:  false,
		},
		{
			name:      "empty string is not reserved",
			stageName: "",
			expected:  false,
		},
		{
			name:      "whitespace only is not reserved",
			stageName: "   ",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkReservedStageName(tt.stageName)
			require.Equal(t, tt.expected, result, "checkReservedStageName(%q) returned unexpected result", tt.stageName)
		})
	}
}

func TestCheckStageNameCasing(t *testing.T) {
	tests := []struct {
		name      string
		stageName string
		expected  bool
	}{
		// Valid stage names (all lowercase)
		{
			name:      "simple lowercase name",
			stageName: "builder",
			expected:  true,
		},
		{
			name:      "lowercase with dash",
			stageName: "builder-base",
			expected:  true,
		},
		{
			name:      "lowercase with underscore",
			stageName: "my_builder",
			expected:  true,
		},
		{
			name:      "lowercase with dot",
			stageName: "builder.v1",
			expected:  true,
		},
		{
			name:      "lowercase with numbers",
			stageName: "builder123",
			expected:  true,
		},
		{
			name:      "complex lowercase name",
			stageName: "my-build_stage.v1",
			expected:  true,
		},
		{
			name:      "starting with underscore",
			stageName: "_builder",
			expected:  true,
		},
		{
			name:      "empty string",
			stageName: "",
			expected:  true, // Empty is handled by other validation
		},
		// Invalid stage names (contains uppercase)
		{
			name:      "all uppercase",
			stageName: "BUILDER",
			expected:  false,
		},
		{
			name:      "mixed case - title case",
			stageName: "Builder",
			expected:  false,
		},
		{
			name:      "mixed case - camel case",
			stageName: "BuilderBase",
			expected:  false,
		},
		{
			name:      "mixed case with dash",
			stageName: "builder-Base",
			expected:  false,
		},
		{
			name:      "mixed case with underscore",
			stageName: "my_Builder",
			expected:  false,
		},
		{
			name:      "single uppercase letter",
			stageName: "buildeR",
			expected:  false,
		},
		{
			name:      "uppercase at start",
			stageName: "Builder-base",
			expected:  false,
		},
		{
			name:      "uppercase in middle",
			stageName: "build-Base-stage",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkStageNameCasing(tt.stageName)
			require.Equal(t, tt.expected, result, "checkStageNameCasing(%q) returned unexpected result", tt.stageName)
		})
	}
}

func TestParseFROM(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string // Rule codes expected to be present
	}{
		// Valid FROM instructions
		{
			name:              "simple valid FROM",
			dockerfileContent: `FROM debian:latest`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with AS stage name",
			dockerfileContent: `FROM debian:latest AS builder`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with platform flag variable",
			dockerfileContent: `FROM --platform=$BUILDPLATFORM debian:latest`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with platform variable and stage name",
			dockerfileContent: `FROM --platform=$BUILDPLATFORM alpine:3.18 AS builder`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with $BUILDPLATFORM (not redundant)",
			dockerfileContent: `FROM --platform=$BUILDPLATFORM alpine AS builder`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with digest",
			dockerfileContent: `FROM alpine@sha256:abc123def456`,
			expectedRules:     []string{},
		},
		{
			name: "FROM with variable",
			dockerfileContent: `ARG CODE_VERSION=1.0
FROM base:${CODE_VERSION}`,
			expectedRules: []string{},
		},
		{
			name:              "no arguments",
			dockerfileContent: `FROM`,
			expectedRules:     []string{"InvalidInstruction"},
		},
		// Invalid FROM instructions
		{
			name:              "FROM with redundant $TARGETPLATFORM",
			dockerfileContent: `FROM --platform=$TARGETPLATFORM alpine AS builder`,
			expectedRules:     []string{"RedundantTargetPlatform"},
		},
		{
			name:              "FROM with redundant $TARGETPLATFORM and no stage",
			dockerfileContent: `FROM --platform=$TARGETPLATFORM debian:latest`,
			expectedRules:     []string{"RedundantTargetPlatform"},
		},
		{
			name:              "FROM with invalid platform OS",
			dockerfileContent: `FROM --platform=invalidOS/amd64 debian:latest`,
			expectedRules:     []string{"FromPlatformFlagConstDisallowed", "FromInvalidPlatform"},
		},
		{
			name:              "FROM with invalid platform arch",
			dockerfileContent: `FROM --platform=linux/invalidarch debian:latest`,
			expectedRules:     []string{"FromPlatformFlagConstDisallowed", "FromInvalidPlatform"},
		},
		{
			name:              "FROM with invalid stage name starting with digit",
			dockerfileContent: `FROM debian:latest AS 1builder`,
			expectedRules:     []string{"FromInvalidStageName"},
		},
		{
			name:              "FROM with invalid stage name with special char",
			dockerfileContent: `FROM debian:latest AS builder@v1`,
			expectedRules:     []string{"FromInvalidStageName"},
		},
		{
			name:              "FROM with mixed casing",
			dockerfileContent: `FROM debian:latest as builder`,
			expectedRules:     []string{"FromAsCasing"},
		},
		{
			name:              "FROM with uppercase stage name",
			dockerfileContent: `FROM alpine AS BuilderBase`,
			expectedRules:     []string{"StageNameCasing"},
		},
		{
			name:              "FROM with all uppercase stage name",
			dockerfileContent: `FROM alpine AS BUILDER`,
			expectedRules:     []string{"StageNameCasing"},
		},
		{
			name:              "FROM with title case stage name",
			dockerfileContent: `FROM debian:latest AS Builder`,
			expectedRules:     []string{"StageNameCasing"},
		},
		{
			name:              "FROM with mixed case stage name with dash",
			dockerfileContent: `FROM ubuntu:22.04 AS builder-Base`,
			expectedRules:     []string{"StageNameCasing"},
		},
		{
			name:              "FROM with reserved stage name 'scratch'",
			dockerfileContent: `FROM alpine AS scratch`,
			expectedRules:     []string{"ReservedStageName"},
		},
		{
			name:              "FROM with reserved stage name 'SCRATCH'",
			dockerfileContent: `FROM alpine AS SCRATCH`,
			expectedRules:     []string{"ReservedStageName"},
		},
		{
			name:              "FROM with reserved stage name 'context'",
			dockerfileContent: `FROM debian:latest AS context`,
			expectedRules:     []string{"ReservedStageName"},
		},
		{
			name:              "FROM with reserved stage name 'CONTEXT'",
			dockerfileContent: `FROM debian:latest AS CONTEXT`,
			expectedRules:     []string{"ReservedStageName"},
		},
		{
			name:              "FROM with reserved stage name 'Context' (mixed case)",
			dockerfileContent: `FROM ubuntu:22.04 AS Context`,
			expectedRules:     []string{"ReservedStageName"},
		},
		// Platform flag tests
		{
			name:              "FROM with constant platform linux/amd64",
			dockerfileContent: `FROM --platform=linux/amd64 alpine`,
			expectedRules:     []string{"FromPlatformFlagConstDisallowed"},
		},
		{
			name:              "FROM with constant platform linux/arm64",
			dockerfileContent: `FROM --platform=linux/arm64 debian:latest`,
			expectedRules:     []string{"FromPlatformFlagConstDisallowed"},
		},
		{
			name:              "FROM with constant platform windows/amd64",
			dockerfileContent: `FROM --platform=windows/amd64 mcr.microsoft.com/windows/servercore:ltsc2019`,
			expectedRules:     []string{"FromPlatformFlagConstDisallowed"},
		},
		{
			name:              "FROM with variable platform $BUILDPLATFORM",
			dockerfileContent: `FROM --platform=$BUILDPLATFORM alpine`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with variable platform $TARGETPLATFORM",
			dockerfileContent: `FROM --platform=$TARGETPLATFORM debian:latest`,
			expectedRules:     []string{"RedundantTargetPlatform"},
		},
		{
			name:              "FROM with variable platform ${BUILDPLATFORM}",
			dockerfileContent: `FROM --platform=${BUILDPLATFORM} alpine`,
			expectedRules:     []string{},
		},
		// Edge cases
		{
			name:              "FROM with lowercase and consistent casing",
			dockerfileContent: `from debian:latest as builder`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with complex valid lowercase stage name",
			dockerfileContent: `FROM node:18 AS my_build-stage.v1`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with lowercase stage name with numbers",
			dockerfileContent: `FROM python:3.11 AS builder123`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with stage name 'scratch-builder' (not reserved)",
			dockerfileContent: `FROM alpine AS scratch-builder`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with stage name 'context-builder' (not reserved)",
			dockerfileContent: `FROM alpine AS context-builder`,
			expectedRules:     []string{},
		},
		{
			name:              "FROM with stage name 'my-scratch' (not reserved)",
			dockerfileContent: `FROM debian:latest AS my-scratch`,
			expectedRules:     []string{},
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

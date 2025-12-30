package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckDuplicateStageNames(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectViolation   bool
		expectedCount     int
	}{
		// Valid cases (no violations)
		{
			name: "single stage",
			dockerfileContent: `FROM alpine AS builder
RUN echo hello`,
			expectViolation: false,
		},
		{
			name: "multiple stages with unique names",
			dockerfileContent: `FROM debian:latest AS deb-builder
RUN apt-get update

FROM golang:latest AS go-builder
RUN go build`,
			expectViolation: false,
		},
		{
			name: "no stage names",
			dockerfileContent: `FROM alpine
FROM debian
FROM ubuntu`,
			expectViolation: false,
		},
		{
			name: "three stages all unique",
			dockerfileContent: `FROM node:18 AS build
RUN npm ci

FROM nginx:alpine AS runtime
COPY --from=build /app /usr/share/nginx/html

FROM scratch AS final
COPY --from=runtime /usr/share/nginx/html /`,
			expectViolation: false,
		},
		// Invalid cases (violations)
		{
			name: "duplicate stage name - exact match",
			dockerfileContent: `FROM debian:latest AS builder
RUN apt-get update

FROM golang:latest AS builder
RUN go build`,
			expectViolation: true,
			expectedCount:   2, // Both occurrences are reported
		},
		{
			name: "duplicate stage name - case insensitive",
			dockerfileContent: `FROM debian:latest AS builder
RUN apt-get update

FROM golang:latest AS BUILDER
RUN go build`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "duplicate stage name - mixed case",
			dockerfileContent: `FROM debian:latest AS builder
RUN apt-get update

FROM golang:latest AS Builder
RUN go build`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "three stages with one duplicate",
			dockerfileContent: `FROM node:18 AS build
RUN npm ci

FROM golang:latest AS build
RUN go build

FROM nginx:alpine AS runtime
COPY --from=build /app /usr/share/nginx/html`,
			expectViolation: true,
			expectedCount:   2,
		},
		{
			name: "three stages with two pairs of duplicates",
			dockerfileContent: `FROM node:18 AS build
RUN npm ci

FROM golang:latest AS build
RUN go build

FROM nginx:alpine AS runtime
COPY --from=build /app /usr/share/nginx/html

FROM scratch AS runtime
COPY --from=runtime /usr/share/nginx/html /`,
			expectViolation: true,
			expectedCount:   4, // 2 for 'build', 2 for 'runtime'
		},
		{
			name: "three identical stage names",
			dockerfileContent: `FROM debian AS base
FROM ubuntu AS base
FROM alpine AS base`,
			expectViolation: true,
			expectedCount:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			if !tt.expectViolation {
				// Filter out only DuplicateStageName rules
				duplicateRules := []Rule{}
				for _, rule := range result.Rules {
					if rule.Code == "DuplicateStageName" {
						duplicateRules = append(duplicateRules, rule)
					}
				}
				require.Empty(t, duplicateRules, "Expected no duplicate stage name violations but got: %v", duplicateRules)
			} else {
				// Filter out only DuplicateStageName rules
				duplicateRules := []Rule{}
				for _, rule := range result.Rules {
					if rule.Code == "DuplicateStageName" {
						duplicateRules = append(duplicateRules, rule)
					}
				}
				require.Len(t, duplicateRules, tt.expectedCount,
					"Expected %d duplicate stage name violations but got %d: %v", tt.expectedCount, len(duplicateRules), duplicateRules)

				// Verify all rules have the correct code and structure
				for _, rule := range duplicateRules {
					require.Equal(t, "DuplicateStageName", rule.Code,
						"Expected rule code 'DuplicateStageName' but got '%s'", rule.Code)
					require.NotEmpty(t, rule.Description, "Rule description should not be empty")
					require.Contains(t, rule.Description, "Duplicate stage name",
						"Description should mention duplicate stage name")
					require.Equal(t, "https://docs.docker.com/reference/build-checks/duplicate-stage-name/",
						rule.Url, "Rule URL should match documentation")
				}
			}
		})
	}
}

func TestParseDockerfileWithDuplicateStageNames(t *testing.T) {
	tests := []struct {
		name                   string
		dockerfileContent      string
		expectedDuplicateRules int
		expectedTotalRules     int
	}{
		{
			name: "valid unique stage names",
			dockerfileContent: `FROM alpine AS build
FROM nginx AS runtime`,
			expectedDuplicateRules: 0,
			expectedTotalRules:     0,
		},
		{
			name: "duplicate stage names",
			dockerfileContent: `FROM debian AS builder
FROM golang AS builder`,
			expectedDuplicateRules: 2,
			expectedTotalRules:     2,
		},
		{
			name: "duplicate with other violations",
			dockerfileContent: `FROM debian:latest as builder
FROM golang:latest as builder
WORKDIR app`,
			expectedDuplicateRules: 2,
			expectedTotalRules:     5, // 2 DuplicateStageName + 2 FromAsCasing + 1 WorkdirRelativePath
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			// Count DuplicateStageName rules
			duplicateCount := 0
			for _, rule := range result.Rules {
				if rule.Code == "DuplicateStageName" {
					duplicateCount++
				}
			}

			require.Equal(t, tt.expectedDuplicateRules, duplicateCount,
				"Expected %d DuplicateStageName rules but got %d", tt.expectedDuplicateRules, duplicateCount)

			require.Equal(t, tt.expectedTotalRules, len(result.Rules),
				"Expected %d total rules but got %d: %v", tt.expectedTotalRules, len(result.Rules), result.Rules)
		})
	}
}

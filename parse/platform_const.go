package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkPlatformFlagConstDisallowed checks if FROM instructions use constant platform values inappropriately.
// Constant platform values (e.g., "linux/amd64") are allowed when:
// 1. The FROM instruction has a stage name (AS <name>)
// 2. The stage is referenced in another FROM instruction (multi-stage build pattern)
//
// This allows the valid pattern:
//
//	FROM --platform=linux/amd64 alpine AS build_amd64
//	FROM --platform=linux/arm64 alpine AS build_arm64
//	FROM build_${TARGETARCH} AS build
//
// But disallows standalone constant platforms:
//
//	FROM --platform=linux/amd64 alpine
func checkPlatformFlagConstDisallowed(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule

	// First pass: collect all stage names defined in FROM instructions
	stageNames := make(map[string]bool)
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) != "FROM" {
			continue
		}

		_, stageName, _ := extractFromComponents(child)
		if stageName != "" {
			// Normalize to lowercase for case-insensitive comparison
			stageNames[strings.ToLower(stageName)] = true
		}
	}

	// Second pass: collect all stage references (FROM <stage-name> or FROM <stage>${VAR})
	referencedStages := make(map[string]bool)
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) != "FROM" {
			continue
		}

		imageRef, _, _ := extractFromComponents(child)
		if imageRef == "" {
			continue
		}

		// Check if the image reference refers to a stage name
		// It could be an exact match or contain variables like build_${TARGETARCH}
		imageRefLower := strings.ToLower(imageRef)

		// Check for exact stage name match
		if stageNames[imageRefLower] {
			referencedStages[imageRefLower] = true
			continue
		}

		// Check if the image reference contains a stage name prefix
		// For example: build_${TARGETARCH} references stages like build_amd64, build_arm64
		for stageName := range stageNames {
			// If the image ref starts with a stage name prefix (before a variable)
			// Example: build_${TARGETARCH} matches build_amd64 and build_arm64
			if strings.HasPrefix(imageRefLower, extractStagePrefix(stageName)) {
				referencedStages[stageName] = true
			}
		}
	}

	// Third pass: check for constant platform flags in FROM instructions
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) != "FROM" {
			continue
		}

		_, stageName, platformFlag := extractFromComponents(child)

		// Skip if no platform flag or platform is a variable
		if platformFlag == "" || !checkPlatformConstant(platformFlag) {
			continue
		}

		// Allow constant platforms if:
		// 1. There's a stage name AND
		// 2. The stage is referenced by another FROM instruction
		if stageName != "" && referencedStages[strings.ToLower(stageName)] {
			continue
		}

		// Flag as violation
		rules = append(rules, Rule{
			StartLine:   child.StartLine,
			EndLine:     child.EndLine,
			Code:        "FromPlatformFlagConstDisallowed",
			Description: "FROM --platform should not use a constant value '" + platformFlag + "'. Use a variable like $BUILDPLATFORM or $TARGETPLATFORM, or specify --platform at build time instead.",
			Url:         "https://docs.docker.com/reference/build-checks/from-platform-flag-const-disallowed/",
			Severity:    SeverityWarning,
		})
	}

	return rules
}

// extractStagePrefix extracts the prefix of a stage name before any underscore or hyphen.
// This helps match stage names like "build_amd64" and "build_arm64" with references like "build_${TARGETARCH}".
// Examples:
//   - "build_amd64" -> "build"
//   - "build-amd64" -> "build"
//   - "builder" -> "builder" (no separator)
func extractStagePrefix(stageName string) string {
	// Find the position of _ or -
	for i, c := range stageName {
		if c == '_' || c == '-' {
			return stageName[:i]
		}
	}
	// No separator found, return the whole name
	return stageName
}

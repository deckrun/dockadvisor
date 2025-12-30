package parse

import (
	"regexp"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseFROM(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "FROM requires at least one argument")}
	}

	// Extract image reference and stage name from the instruction
	// Format: FROM [--platform=<platform>] <image>[:<tag>|@<digest>] [AS <name>]
	imageRef, stageName, platformFlag := extractFromComponents(node)

	// ERROR CHECKS - Return immediately on first error
	// Validate image reference is not empty
	if strings.TrimSpace(imageRef) == "" {
		return []Rule{NewErrorRule(node, "FromMissingImage",
			"FROM instruction must specify an image reference",
			"https://docs.docker.com/reference/dockerfile/#from")}
	}

	// Validate image reference format
	if imageRef != "" && !checkImageReferenceFormat(imageRef) {
		return []Rule{NewErrorRule(node, "FromInvalidImageReference",
			"FROM instruction has invalid image reference format: '"+imageRef+"'",
			"https://docs.docker.com/reference/dockerfile/#from")}
	}

	// Validate --platform flag format if present
	// Note: Platform flag constant check is now handled globally in checkPlatformFlagConstDisallowed
	// to allow constant platforms in multi-stage builds where stages are referenced
	if platformFlag != "" && !checkPlatformFormat(platformFlag) {
		return []Rule{NewErrorRule(node, "FromInvalidPlatform",
			"FROM instruction has invalid --platform flag format: '"+platformFlag+"'",
			"https://docs.docker.com/reference/dockerfile/#from")}
	}

	// Validate stage name format if present
	if stageName != "" && !checkStageNameFormat(stageName) {
		return []Rule{NewErrorRule(node, "FromInvalidStageName",
			"FROM instruction AS stage name is invalid: '"+stageName+"'. Stage names must start with a letter or underscore and contain only alphanumeric characters, underscores, hyphens, and dots.",
			"https://docs.docker.com/reference/dockerfile/#from")}
	}

	// Check if stage name is a reserved word
	if stageName != "" && checkReservedStageName(stageName) {
		return []Rule{NewErrorRule(node, "ReservedStageName",
			"'"+stageName+"' is reserved and should not be used as a stage name",
			"https://docs.docker.com/reference/build-checks/reserved-stage-name/")}
	}

	// WARNING CHECKS - Collect warnings
	var fromRules []Rule

	// Check if platform flag is redundant (using $TARGETPLATFORM)
	if platformFlag != "" && checkRedundantTargetPlatform(platformFlag) {
		fromRules = append(fromRules, NewWarningRule(node, "RedundantTargetPlatform",
			"Setting platform to predefined $TARGETPLATFORM in FROM is redundant as this is the default behavior",
			"https://docs.docker.com/reference/build-checks/redundant-target-platform/"))
	}

	// Check if stage name uses lowercase casing
	if stageName != "" && !checkStageNameCasing(stageName) {
		fromRules = append(fromRules, NewWarningRule(node, "StageNameCasing",
			"Stage name '"+stageName+"' should be lowercase",
			"https://docs.docker.com/reference/build-checks/stage-name-casing/"))
	}

	// While Dockerfile keywords can be either uppercase or lowercase, mixing case styles is not recommended for readability. This rule reports violations where mixed case style occurs for a FROM instruction with an AS keyword declaring a stage name.
	// node.Original contains the full original line, e.g., "FROM debian:latest as builder"
	if !checkFromAsCasing(node.Value, node.Original) {
		fromRules = append(fromRules, NewWarningRule(node, "FromAsCasing",
			"FROM instruction with AS keyword uses inconsistent casing. Ensure that both FROM and AS keywords use the same casing style (either both uppercase or both lowercase) for better readability.",
			"https://docs.docker.com/reference/build-checks/from-as-casing/"))
	}

	return fromRules
}

// extractFromComponents extracts the image reference, stage name, and platform flag from a FROM instruction
// Returns: (imageRef, stageName, platformFlag)
func extractFromComponents(node *parser.Node) (string, string, string) {
	var imageRef, stageName, platformFlag string

	// Check for --platform flag in node.Flags
	for _, flag := range node.Flags {
		if strings.HasPrefix(flag, "--platform=") {
			platformFlag = strings.TrimPrefix(flag, "--platform=")
			break
		}
	}

	// Get arguments from node.Next
	current := node.Next
	if current == nil {
		return "", "", platformFlag
	}

	// Get image reference (first argument)
	imageRef = current.Value
	current = current.Next

	// Check for AS stage name
	if current != nil && strings.ToUpper(current.Value) == "AS" {
		current = current.Next
		if current != nil {
			stageName = current.Value
		}
	}

	return imageRef, stageName, platformFlag
}

// checkImageReferenceFormat validates the image reference format
// Valid formats: <image>, <image>:<tag>, <image>@<digest>, or with registry/repository prefixes
// Also allows variables like ${VAR} or $VAR in the image reference
func checkImageReferenceFormat(imageRef string) bool {
	// Basic validation: image reference should not be empty after trimming
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return false
	}

	// Check if it contains a variable (e.g., ${CODE_VERSION} or $VERSION)
	// Variables should be in proper format: $VAR or ${VAR} or ${VAR:-default}
	if strings.Contains(imageRef, "$") {
		// Allow variables with proper format:
		// - ${VAR} - simple variable
		// - ${VAR:-default} - variable with default value (Dockerfile supported)
		// - $VAR - simple variable without braces
		varPattern := regexp.MustCompile(`\$\{[a-zA-Z_][a-zA-Z0-9_]*(?::-[^}]*)?\}|\$[a-zA-Z_][a-zA-Z0-9_]*`)
		if varPattern.MatchString(imageRef) {
			return true
		}
		// If $ is present but not in valid variable format, it's invalid
		return false
	}

	// Image reference pattern:
	// - Can have registry (domain.com:port/)
	// - Can have repository path (namespace/image)
	// - Can have tag (:tag) or digest (@sha256:...)
	// - Name components can contain lowercase letters, digits, and separators (.-_)

	// Simple validation: check for obviously invalid characters
	// Docker image names can't contain spaces or most special chars (except :@./_-)
	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9:@./_-]`)
	if invalidChars.MatchString(imageRef) {
		return false
	}

	// Check that we don't have multiple @ or improper use of :
	atCount := strings.Count(imageRef, "@")
	if atCount > 1 {
		return false
	}

	// If there's an @, it should be followed by a digest (like sha256:...)
	if atCount == 1 {
		parts := strings.Split(imageRef, "@")
		if len(parts) != 2 || !strings.HasPrefix(parts[1], "sha256:") && !strings.HasPrefix(parts[1], "sha512:") {
			// Allow other digest formats too
			if !regexp.MustCompile(`^[a-z0-9]+:[a-f0-9]+$`).MatchString(parts[1]) {
				return false
			}
		}
	}

	return true
}

// checkRedundantTargetPlatform checks if the platform flag is set to $TARGETPLATFORM.
// Returns true if the platform is $TARGETPLATFORM (redundant), false otherwise.
// Setting platform to $TARGETPLATFORM is redundant since that's the default behavior.
// Examples:
//   - "$TARGETPLATFORM" -> true (redundant)
//   - "linux/amd64" -> false (not redundant)
//   - "$BUILDPLATFORM" -> false (not redundant)
func checkRedundantTargetPlatform(platform string) bool {
	platform = strings.TrimSpace(platform)
	return platform == "$TARGETPLATFORM"
}

// checkPlatformConstant checks if the platform flag uses a constant value.
// Returns true if platform is a constant (not a variable), false if it's a variable.
// Constant values like "linux/amd64" are disallowed to enable multi-platform builds.
// Variables like "$BUILDPLATFORM" or "${TARGETPLATFORM}" are allowed.
func checkPlatformConstant(platform string) bool {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return false
	}

	// If it starts with $, it's a variable reference, not a constant
	return !strings.HasPrefix(platform, "$")
}

// checkPlatformFormat validates the --platform flag format
// Valid formats: linux/amd64, linux/arm64, windows/amd64, etc.
func checkPlatformFormat(platform string) bool {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return false
	}

	// Allow variables like $BUILDPLATFORM
	if strings.HasPrefix(platform, "$") {
		return true
	}

	// Platform format: <os>[/<arch>[/<variant>]]
	// Examples: linux/amd64, linux/arm64/v8, windows/amd64
	parts := strings.Split(platform, "/")
	if len(parts) < 1 || len(parts) > 3 {
		return false
	}

	// Validate OS (first part)
	validOS := map[string]bool{
		"linux":   true,
		"windows": true,
		"darwin":  true,
		"freebsd": true,
	}

	if !validOS[parts[0]] {
		return false
	}

	// If arch is specified, validate it
	if len(parts) >= 2 {
		validArch := map[string]bool{
			"amd64":   true,
			"arm64":   true,
			"arm":     true,
			"386":     true,
			"ppc64le": true,
			"s390x":   true,
			"riscv64": true,
		}
		if !validArch[parts[1]] {
			return false
		}
	}

	return true
}

// checkStageNameFormat validates the stage name format
// Stage names must start with a letter or underscore and contain only alphanumeric characters, underscores, hyphens, and dots
func checkStageNameFormat(stageName string) bool {
	stageName = strings.TrimSpace(stageName)
	if stageName == "" {
		return false
	}

	// Stage name pattern: must start with [a-zA-Z_] and contain only [a-zA-Z0-9_.-]
	pattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.-]*$`)
	return pattern.MatchString(stageName)
}

// checkReservedStageName checks if the stage name is a reserved word.
// Reserved words should not be used as stage names in multi-stage builds.
// Returns true if the stage name is reserved, false otherwise.
// Reserved words are:
//   - "scratch" (case-insensitive)
//   - "context" (case-insensitive)
//
// Examples:
//   - "scratch" -> true (reserved)
//   - "SCRATCH" -> true (reserved, case-insensitive)
//   - "context" -> true (reserved)
//   - "Context" -> true (reserved, case-insensitive)
//   - "builder" -> false (not reserved)
func checkReservedStageName(stageName string) bool {
	stageName = strings.TrimSpace(stageName)
	if stageName == "" {
		return false // Empty stage name is handled by other validation
	}

	// Convert to lowercase for case-insensitive comparison
	lowerStageName := strings.ToLower(stageName)

	// Reserved words
	reservedWords := map[string]bool{
		"scratch": true,
		"context": true,
	}

	return reservedWords[lowerStageName]
}

// checkStageNameCasing checks if the stage name is all lowercase.
// Returns true if the stage name is all lowercase, false if it contains any uppercase letters.
// To help distinguish Dockerfile instruction keywords from identifiers, stage names should be lowercase.
// Examples:
//   - "builder" -> true (all lowercase)
//   - "builder-base" -> true (all lowercase with hyphen)
//   - "BuilderBase" -> false (contains uppercase)
//   - "BUILDER" -> false (all uppercase)
func checkStageNameCasing(stageName string) bool {
	stageName = strings.TrimSpace(stageName)
	if stageName == "" {
		return true // Empty stage name is handled by other validation
	}

	// Check if the stage name is all lowercase
	// Stage names can contain lowercase letters, numbers, underscores, hyphens, and dots
	// Any uppercase letter makes it invalid
	return stageName == strings.ToLower(stageName)
}

// checkFromAsCasing checks if the FROM instruction with AS keyword uses consistent casing.
// Returns true if the casing is consistent, false otherwise.
// While Dockerfile keywords can be either uppercase or lowercase, mixing case styles is not
// recommended for readability. This rule reports violations where mixed case style occurs
// for a FROM instruction with an AS keyword declaring a stage name.
func checkFromAsCasing(fromKeyword, originalLine string) bool {
	// Split the original line to find both FROM and AS keywords
	// Example: "FROM debian:latest as builder"
	parts := strings.Fields(originalLine)

	// If there are less than 4 parts, there's no AS keyword
	// Format: FROM <image> AS <stage-name>
	if len(parts) < 4 {
		return true
	}

	// Find the AS keyword (should be at index 2 typically)
	var asKeywordIndex = -1
	for i, part := range parts {
		if strings.ToUpper(part) == "AS" {
			asKeywordIndex = i
			break
		}
	}

	// If no AS keyword found, it's valid
	if asKeywordIndex == -1 {
		return true
	}

	asKeyword := parts[asKeywordIndex]

	// Check if AS keyword casing matches FROM keyword casing
	// Both should be uppercase or both should be lowercase
	fromIsUpper := fromKeyword == strings.ToUpper(fromKeyword)
	asIsUpper := asKeyword == "AS"

	// If FROM is uppercase and AS is uppercase, or FROM is lowercase and AS is lowercase
	return fromIsUpper == asIsUpper
}

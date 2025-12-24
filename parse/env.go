package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseENV(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "ENV requires at least one argument")}
	}

	// Extract env configuration
	envConfig := extractENVConfig(node)

	// ERROR CHECKS - Return immediately on first error

	// Validate that config is not empty
	if strings.TrimSpace(envConfig) == "" {
		return []Rule{NewErrorRule(node, "EnvMissingKeyValue",
			"ENV instruction must specify at least one key=value pair",
			"https://docs.docker.com/reference/dockerfile/#env")}
	}

	// Validate env key=value format (if not using legacy syntax)
	if !checkENVLegacySyntax(envConfig) && !checkENVFormat(envConfig) {
		return []Rule{NewErrorRule(node, "EnvInvalidFormat",
			"ENV instruction must be in the format <key>=<value> [<key>=<value>...]",
			"https://docs.docker.com/reference/dockerfile/#env")}
	}

	// WARNING CHECKS - Accumulate warnings and return at end
	var envRules []Rule

	// Check if using legacy syntax (space-separated, discouraged)
	if checkENVLegacySyntax(envConfig) {
		envRules = append(envRules, NewWarningRule(node, "LegacyKeyValueFormat",
			"Legacy key/value format with whitespace separator should not be used. Use ENV key=value format instead",
			"https://docs.docker.com/reference/build-checks/legacy-key-value-format/"))
	}

	return envRules
}

// extractENVConfig extracts the env configuration from the ENV instruction
func extractENVConfig(node *parser.Node) string {
	if node.Next == nil {
		return ""
	}

	// Use the Original field and strip the instruction keyword
	// This preserves the exact format including key=value pairs
	config := strings.TrimPrefix(node.Original, node.Value)
	return strings.TrimSpace(config)
}

// checkENVLegacySyntax checks if using the legacy ENV <key> <value> syntax
func checkENVLegacySyntax(config string) bool {
	config = strings.TrimSpace(config)

	if config == "" {
		return false
	}

	// Legacy syntax: ENV <key> <value>
	// This means no '=' sign and at least one space
	// It's legacy if there's no '=' and there are spaces
	hasEquals := strings.Contains(config, "=")
	hasSpace := strings.Contains(config, " ")

	// Legacy syntax has spaces but no equals
	return hasSpace && !hasEquals
}

// checkENVFormat validates the ENV instruction format
func checkENVFormat(config string) bool {
	config = strings.TrimSpace(config)

	if config == "" {
		return false
	}

	// ENV accepts: <key>=<value> [<key>=<value>...]
	// Must contain at least one '=' sign
	if !strings.Contains(config, "=") {
		return false
	}

	// Split by spaces but be careful with quoted strings
	// Basic validation: check that we have at least one key=value pair
	// This is a simplified check - the Docker parser handles complex quoting
	parts := strings.Fields(config)

	// Check if at least one part contains '='
	hasValidPair := false
	for _, part := range parts {
		if strings.Contains(part, "=") {
			// Check that key is not empty (not starting with =)
			if !strings.HasPrefix(part, "=") {
				hasValidPair = true
				break
			}
		}
	}

	return hasValidPair
}

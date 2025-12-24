package parse

import (
	"regexp"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseARG(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "ARG requires at least one argument")}
	}

	// Extract arg configuration
	argConfig := extractARGConfig(node)

	// ERROR CHECKS - Return immediately on first error

	// Validate that config is not empty
	if strings.TrimSpace(argConfig) == "" {
		return []Rule{NewErrorRule(node, "ArgMissingName",
			"ARG instruction must specify at least one argument name",
			"https://docs.docker.com/reference/dockerfile/#arg")}
	}

	// Validate ARG format (if not using legacy syntax)
	if !checkARGLegacySyntax(argConfig) && !checkARGFormat(argConfig) {
		return []Rule{NewErrorRule(node, "ArgInvalidFormat",
			"ARG instruction must be in the format <name>[=<default value>]",
			"https://docs.docker.com/reference/dockerfile/#arg")}
	}

	// WARNING CHECKS - Accumulate warnings and return at end
	var argRules []Rule

	// Check if using legacy syntax (space-separated, deprecated)
	if checkARGLegacySyntax(argConfig) {
		argRules = append(argRules, NewWarningRule(node, "LegacyKeyValueFormat",
			"Legacy key/value format with whitespace separator should not be used. Use ARG key=value format instead",
			"https://docs.docker.com/reference/build-checks/legacy-key-value-format/"))
	}

	return argRules
}

// extractARGConfig extracts the arg configuration from the ARG instruction
func extractARGConfig(node *parser.Node) string {
	if node.Next == nil {
		return ""
	}

	// Collect all arguments into a single string
	var parts []string
	current := node.Next
	for current != nil {
		parts = append(parts, current.Value)
		current = current.Next
	}

	return strings.Join(parts, " ")
}

// checkARGLegacySyntax checks if using the legacy ARG <key> <value> syntax
func checkARGLegacySyntax(config string) bool {
	config = strings.TrimSpace(config)

	if config == "" {
		return false
	}

	// Check if there are multiple space-separated parts
	parts := strings.Fields(config)
	if len(parts) <= 1 {
		return false // Single arg, not legacy
	}

	// Legacy/ambiguous syntax occurs when:
	// - Multiple parts exist (separated by spaces)
	// - At least one part doesn't have an equals sign
	// This is discouraged because it's ambiguous whether you mean:
	//   - Multiple args without defaults: ARG foo bar (defines $foo and $bar)
	//   - Legacy attempt at key=value: ARG foo bar (trying to set $foo=bar)
	// Docker recommends using = explicitly: ARG foo=bar or separate lines
	for _, part := range parts {
		if !strings.Contains(part, "=") {
			return true // Found a part without equals - ambiguous/legacy
		}
	}

	return false // All parts have equals - valid multi-arg format
}

// checkARGFormat validates the ARG instruction format
func checkARGFormat(config string) bool {
	config = strings.TrimSpace(config)

	if config == "" {
		return false
	}

	// ARG accepts: <name>[=<default value>] [<name>[=<default value>]...]
	// Valid arg name: alphanumeric, underscore
	// Can optionally have =value

	// Split by spaces to handle multiple ARGs
	parts := strings.Fields(config)

	for _, part := range parts {
		// Each part should be name or name=value
		// Name must start with letter or underscore
		// Name can contain letters, numbers, underscore

		// Check if it has '='
		if strings.Contains(part, "=") {
			// Split by first '='
			nameValue := strings.SplitN(part, "=", 2)
			if len(nameValue) != 2 {
				return false
			}
			// Validate name part
			if !isValidARGName(nameValue[0]) {
				return false
			}
			// Value can be anything (including empty)
		} else {
			// Just a name, validate it
			if !isValidARGName(part) {
				return false
			}
		}
	}

	return true
}

// isValidARGName validates that an ARG name follows naming rules
func isValidARGName(name string) bool {
	if name == "" {
		return false
	}

	// ARG names must start with letter or underscore
	// and contain only letters, numbers, underscore
	pattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return pattern.MatchString(name)
}

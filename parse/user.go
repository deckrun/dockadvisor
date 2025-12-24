package parse

import (
	"regexp"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseUSER(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "USER requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract user configuration
	userConfig := extractUSERConfig(node)

	// Validate that config is not empty
	if strings.TrimSpace(userConfig) == "" {
		return []Rule{NewErrorRule(node, "UserMissingValue",
			"USER instruction must specify a user",
			"https://docs.docker.com/reference/dockerfile/#user")}
	}

	// Validate user format: <user>[:<group>] or <UID>[:<GID>]
	if !checkUSERFormat(userConfig) {
		return []Rule{NewErrorRule(node, "UserInvalidFormat",
			"USER instruction must be in the format <user>[:<group>] or <UID>[:<GID>]",
			"https://docs.docker.com/reference/dockerfile/#user")}
	}

	// No warnings in this file
	return nil
}

// extractUSERConfig extracts the user configuration from the USER instruction
func extractUSERConfig(node *parser.Node) string {
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

// checkUSERFormat validates the USER instruction format
func checkUSERFormat(config string) bool {
	config = strings.TrimSpace(config)

	if config == "" {
		return false
	}

	// USER accepts: <user>[:<group>] or <UID>[:<GID>]
	// Valid characters: alphanumeric, underscore, dash, dot
	// UIDs/GIDs are numeric
	// Can have optional :group or :GID after the user/UID

	// Pattern allows:
	// - Username: letters, numbers, underscore, dash, dot, dollar (for variables)
	// - UID: numbers
	// - Optional group/GID after colon with same rules
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_.$-]+(?::[a-zA-Z0-9_.$-]+)?$`)

	return pattern.MatchString(config)
}

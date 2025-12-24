package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseLABEL(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "LABEL requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract label configuration
	labelConfig := extractLABELConfig(node)

	// Validate that config is not empty
	if strings.TrimSpace(labelConfig) == "" {
		return []Rule{NewErrorRule(node, "LabelMissingKeyValue",
			"LABEL instruction must specify at least one key=value pair",
			"https://docs.docker.com/reference/dockerfile/#label")}
	}

	// Validate label key=value format
	if !checkLABELFormat(labelConfig) {
		return []Rule{NewErrorRule(node, "LabelInvalidFormat",
			"LABEL instruction must be in the format <key>=<value> [<key>=<value>...]",
			"https://docs.docker.com/reference/dockerfile/#label")}
	}

	// No warnings in this file
	return nil
}

// extractLABELConfig extracts the label configuration from the LABEL instruction
func extractLABELConfig(node *parser.Node) string {
	if node.Next == nil {
		return ""
	}

	// Use the Original field and strip the instruction keyword
	// This preserves the exact format including key=value pairs
	config := strings.TrimPrefix(node.Original, node.Value)
	return strings.TrimSpace(config)
}

// checkLABELFormat validates the LABEL instruction format
func checkLABELFormat(config string) bool {
	config = strings.TrimSpace(config)

	if config == "" {
		return false
	}

	// LABEL accepts: <key>=<value> [<key>=<value>...]
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

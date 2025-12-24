package parse

import (
	"encoding/json"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseSHELL(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "SHELL requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract shell configuration
	shellConfig := extractSHELLConfig(node)

	// Validate that config is not empty
	if strings.TrimSpace(shellConfig) == "" {
		return []Rule{NewErrorRule(node, "ShellMissingConfig",
			"SHELL instruction must specify a shell configuration",
			"https://docs.docker.com/reference/dockerfile/#shell")}
	}

	// SHELL instruction MUST be in JSON form
	if !strings.HasPrefix(strings.TrimSpace(shellConfig), "[") {
		return []Rule{NewErrorRule(node, "ShellRequiresJsonForm",
			"SHELL instruction must be written in JSON form (e.g., SHELL [\"executable\", \"parameters\"])",
			"https://docs.docker.com/reference/dockerfile/#shell")}
	}

	// Validate JSON form
	if !checkSHELLJsonForm(shellConfig) {
		return []Rule{NewErrorRule(node, "ShellInvalidJsonForm",
			"SHELL instruction must be a valid JSON array with double quotes",
			"https://docs.docker.com/reference/dockerfile/#shell")}
	}

	// No warnings in this file
	return nil
}

// extractSHELLConfig extracts the shell configuration from the SHELL instruction
func extractSHELLConfig(node *parser.Node) string {
	if node.Next == nil {
		return ""
	}

	// Use the Original field and strip the instruction keyword
	// This preserves the exact format including key=value pairs
	config := strings.TrimPrefix(node.Original, node.Value)
	return strings.TrimSpace(config)
}

// checkSHELLJsonForm validates that the SHELL instruction is valid JSON array
func checkSHELLJsonForm(config string) bool {
	config = strings.TrimSpace(config)

	// Must start with '[' and end with ']'
	if !strings.HasPrefix(config, "[") || !strings.HasSuffix(config, "]") {
		return false
	}

	// Try to parse as JSON array
	var arr []string
	if err := json.Unmarshal([]byte(config), &arr); err != nil {
		return false
	}

	// Array should not be empty
	if len(arr) == 0 {
		return false
	}

	// Check that original string uses double quotes, not single quotes
	// Single quotes are not valid JSON
	if strings.Contains(config, "['") || strings.Contains(config, "']") {
		return false
	}

	return true
}

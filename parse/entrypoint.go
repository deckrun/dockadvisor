package parse

import (
	"encoding/json"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseENTRYPOINT(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "ENTRYPOINT requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract command - use Original field for accurate detection
	trimmedOriginal := strings.TrimSpace(strings.TrimPrefix(node.Original, node.Value))
	command := extractENTRYPOINTCommand(node)
	if command == "" {
		command = trimmedOriginal
	}

	// Validate that command is not empty
	if strings.TrimSpace(command) == "" {
		return []Rule{NewErrorRule(node, "EntrypointMissingCommand",
			"ENTRYPOINT instruction must specify a command to execute",
			"https://docs.docker.com/reference/dockerfile/#entrypoint")}
	}

	// Check if it's exec form (starts with '[') - use Original field for accurate detection
	isExecForm := strings.HasPrefix(strings.TrimSpace(trimmedOriginal), "[")
	if isExecForm {
		// Validate JSON format using trimmedOriginal to preserve exact format
		if !checkENTRYPOINTExecFormJSON(trimmedOriginal) {
			return []Rule{NewErrorRule(node, "EntrypointInvalidExecForm",
				"ENTRYPOINT exec form must be a valid JSON array with double quotes",
				"https://docs.docker.com/reference/dockerfile/#entrypoint")}
		}
	}
	// Note: JSONArgsRecommended check is now handled globally in checkJSONArgsRecommended
	// to suppress the warning when a SHELL instruction is explicitly set

	// No warnings in this file
	return nil
}

// extractENTRYPOINTCommand extracts the command string from the ENTRYPOINT instruction
func extractENTRYPOINTCommand(node *parser.Node) string {
	if node.Next == nil {
		return ""
	}

	// Collect all arguments into a single command string
	var parts []string
	current := node.Next
	for current != nil {
		parts = append(parts, current.Value)
		current = current.Next
	}

	return strings.Join(parts, " ")
}

// checkENTRYPOINTExecFormJSON validates that the exec form is valid JSON array
func checkENTRYPOINTExecFormJSON(command string) bool {
	command = strings.TrimSpace(command)

	// Must start with '[' and end with ']'
	if !strings.HasPrefix(command, "[") || !strings.HasSuffix(command, "]") {
		return false
	}

	// Try to parse as JSON array
	var arr []string
	if err := json.Unmarshal([]byte(command), &arr); err != nil {
		return false
	}

	// Array should not be empty for ENTRYPOINT (unlike CMD)
	if len(arr) == 0 {
		return false
	}

	// Check that original string uses double quotes, not single quotes
	// Single quotes are not valid JSON
	if strings.Contains(command, "['") || strings.Contains(command, "']") {
		return false
	}

	return true
}

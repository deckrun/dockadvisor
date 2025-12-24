package parse

import (
	"encoding/json"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseCMD(node *parser.Node) []Rule {
	// Special case: CMD [] is valid (empty array as default params to ENTRYPOINT)
	// Check Original field to see if it's just "CMD []"
	trimmedOriginal := strings.TrimSpace(strings.TrimPrefix(node.Original, node.Value))
	if node.Next == nil && trimmedOriginal != "[]" {
		return []Rule{invalidInstructionRule(node, "CMD requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract command
	command := extractCMDCommand(node)
	if command == "" {
		command = trimmedOriginal
	}

	// Validate that command is not empty
	if strings.TrimSpace(command) == "" {
		return []Rule{NewErrorRule(node, "CmdMissingCommand",
			"CMD instruction must specify a command to execute",
			"https://docs.docker.com/reference/dockerfile/#cmd")}
	}

	// Check if it's exec form (starts with '[') - use Original field for accurate detection
	isExecForm := strings.HasPrefix(strings.TrimSpace(trimmedOriginal), "[")
	if isExecForm {
		// Validate JSON format using trimmedOriginal to preserve exact format
		if !checkCMDExecFormJSON(trimmedOriginal) {
			return []Rule{NewErrorRule(node, "CmdInvalidExecForm",
				"CMD exec form must be a valid JSON array with double quotes",
				"https://docs.docker.com/reference/dockerfile/#cmd")}
		}
	}
	// Note: JSONArgsRecommended check is now handled globally in checkJSONArgsRecommended
	// to suppress the warning when a SHELL instruction is explicitly set

	// No warnings in this file
	return nil
}

// extractCMDCommand extracts the command string from the CMD instruction
func extractCMDCommand(node *parser.Node) string {
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

// checkCMDExecFormJSON validates that the exec form is valid JSON array
// This is the same validation as RUN, but kept separate for clarity
func checkCMDExecFormJSON(command string) bool {
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

	// Array can be empty for CMD (when used as default params to ENTRYPOINT)
	// But we'll allow it since it's valid according to the docs

	// Check that original string uses double quotes, not single quotes
	// Single quotes are not valid JSON
	if strings.Contains(command, "['") || strings.Contains(command, "']") {
		return false
	}

	return true
}

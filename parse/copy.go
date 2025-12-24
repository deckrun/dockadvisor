package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseCOPY(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "COPY requires at least source and destination arguments")}
	}

	// ERROR CHECKS - Return immediately on first error
	// COPY requires at least 2 arguments: source and destination
	// Count non-flag arguments
	argCount := 0
	current := node.Next
	for current != nil {
		// Skip flags (--from, --chown, --chmod, --link, --parents, --exclude)
		if !strings.HasPrefix(current.Value, "--") {
			argCount++
		}
		current = current.Next
	}

	if argCount < 2 {
		return []Rule{NewErrorRule(node, "CopyMissingArguments",
			"COPY instruction requires at least source and destination arguments",
			"https://docs.docker.com/reference/dockerfile/#copy")}
	}

	// Validate flags if present (flags are stored in node.Flags, not node.Next)
	for _, flag := range node.Flags {
		// Check if it's a valid flag
		flagName := strings.Split(flag, "=")[0]
		if !isValidCOPYFlag(flagName) {
			return []Rule{NewErrorRule(node, "CopyInvalidFlag",
				"COPY instruction has invalid flag: "+flagName,
				"https://docs.docker.com/reference/dockerfile/#copy")}
		}
	}

	// No warnings in this file
	return nil
}

// isValidCOPYFlag checks if a flag is valid for COPY instruction
func isValidCOPYFlag(flag string) bool {
	validFlags := map[string]bool{
		"--from":    true,
		"--chown":   true,
		"--chmod":   true,
		"--link":    true,
		"--parents": true,
		"--exclude": true,
	}

	return validFlags[flag]
}

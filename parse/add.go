package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseADD(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "ADD requires at least source and destination arguments")}
	}

	// ERROR CHECKS - Return immediately on first error
	// ADD requires at least 2 arguments: source and destination
	// Count non-flag arguments
	argCount := 0
	current := node.Next
	for current != nil {
		// Skip flags (--keep-git-dir, --checksum, --chown, --chmod, --link, --exclude)
		if !strings.HasPrefix(current.Value, "--") {
			argCount++
		}
		current = current.Next
	}

	if argCount < 2 {
		return []Rule{NewErrorRule(node, "AddMissingArguments",
			"ADD instruction requires at least source and destination arguments",
			"https://docs.docker.com/reference/dockerfile/#add")}
	}

	// Validate flags if present (flags are stored in node.Flags, not node.Next)
	for _, flag := range node.Flags {
		// Check if it's a valid flag
		flagName := strings.Split(flag, "=")[0]
		if !isValidADDFlag(flagName) {
			return []Rule{NewErrorRule(node, "AddInvalidFlag",
				"ADD instruction has invalid flag: "+flagName,
				"https://docs.docker.com/reference/dockerfile/#add")}
		}
	}

	// No warnings in this file
	return nil
}

// isValidADDFlag checks if a flag is valid for ADD instruction
func isValidADDFlag(flag string) bool {
	validFlags := map[string]bool{
		"--keep-git-dir": true,
		"--checksum":     true,
		"--chown":        true,
		"--chmod":        true,
		"--link":         true,
		"--exclude":      true,
	}

	return validFlags[flag]
}

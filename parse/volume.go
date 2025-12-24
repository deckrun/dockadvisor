package parse

import (
	"encoding/json"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseVOLUME(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "VOLUME requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract volume configuration
	volumeConfig := extractVOLUMEConfig(node)

	// Validate that config is not empty
	if strings.TrimSpace(volumeConfig) == "" {
		return []Rule{NewErrorRule(node, "VolumeMissingPath",
			"VOLUME instruction must specify at least one mount point",
			"https://docs.docker.com/reference/dockerfile/#volume")}
	}

	// If it's JSON form (starts with '['), validate JSON syntax
	if strings.HasPrefix(strings.TrimSpace(volumeConfig), "[") {
		if !checkVOLUMEJsonForm(volumeConfig) {
			return []Rule{NewErrorRule(node, "VolumeInvalidJsonForm",
				"VOLUME JSON form must be a valid JSON array with double quotes",
				"https://docs.docker.com/reference/dockerfile/#volume")}
		}
	}

	// No warnings in this file
	return nil
}

// extractVOLUMEConfig extracts the volume configuration from the VOLUME instruction
func extractVOLUMEConfig(node *parser.Node) string {
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

// checkVOLUMEJsonForm validates that the VOLUME JSON form is valid
func checkVOLUMEJsonForm(config string) bool {
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
	if strings.Contains(config, "['") || strings.Contains(config, "']") {
		return false
	}

	return true
}

package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseHEALTHCHECK(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "HEALTHCHECK requires arguments")}
	}

	// ERROR CHECKS - Return immediately on first error
	// HEALTHCHECK has two forms:
	// 1. HEALTHCHECK [OPTIONS] CMD command
	// 2. HEALTHCHECK NONE

	// Check if it's HEALTHCHECK NONE
	if node.Next != nil && strings.ToUpper(node.Next.Value) == "NONE" {
		// Valid HEALTHCHECK NONE
		return nil
	}

	// Otherwise, must have CMD keyword
	hasCMD := false
	current := node.Next
	for current != nil {
		if strings.ToUpper(current.Value) == "CMD" {
			hasCMD = true
			break
		}
		current = current.Next
	}

	if !hasCMD {
		return []Rule{NewErrorRule(node, "HealthcheckMissingCmd",
			"HEALTHCHECK instruction must include CMD keyword or be HEALTHCHECK NONE",
			"https://docs.docker.com/reference/dockerfile/#healthcheck")}
	}

	// No warnings in this file
	return nil
}

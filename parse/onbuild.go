package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseONBUILD(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "ONBUILD requires an instruction argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// ONBUILD must be followed by another Dockerfile instruction
	// Extract the instruction from the Original field
	config := strings.TrimPrefix(node.Original, node.Value)
	config = strings.TrimSpace(config)

	if config == "" {
		return []Rule{NewErrorRule(node, "OnbuildMissingInstruction",
			"ONBUILD must be followed by a Dockerfile instruction",
			"https://docs.docker.com/reference/dockerfile/#onbuild")}
	}

	// No warnings in this file
	return nil
}

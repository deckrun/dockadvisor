package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseSTOPSIGNAL(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "STOPSIGNAL requires a signal argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract signal value
	signal := strings.TrimSpace(node.Next.Value)

	if signal == "" {
		return []Rule{NewErrorRule(node, "StopsignalMissingValue",
			"STOPSIGNAL must specify a signal",
			"https://docs.docker.com/reference/dockerfile/#stopsignal")}
	}

	// No warnings in this file
	return nil
}

package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseMAINTAINER(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "MAINTAINER requires a name argument")}
	}

	// ERROR CHECKS - Return immediately on first error

	// Validate that a name is provided
	name := strings.TrimSpace(node.Next.Value)
	if name == "" {
		return []Rule{NewErrorRule(node, "MaintainerMissingName",
			"MAINTAINER must specify a name",
			"https://docs.docker.com/reference/dockerfile/#maintainer-deprecated")}
	}

	// WARNING CHECKS - Accumulate warnings and return at end
	var maintainerRules []Rule

	// MAINTAINER is deprecated - warn users to use LABEL instead
	maintainerRules = append(maintainerRules, NewWarningRule(node, "MaintainerDeprecated",
		"MAINTAINER instruction is deprecated in favor of using label",
		"https://docs.docker.com/reference/build-checks/maintainer-deprecated/"))

	return maintainerRules
}

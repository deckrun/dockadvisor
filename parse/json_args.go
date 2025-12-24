package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkJSONArgsRecommended checks if CMD or ENTRYPOINT use shell form without an explicit SHELL instruction.
// According to Docker best practices, using shell form prevents proper OS signal handling (SIGTERM, SIGKILL)
// because the process runs as a child of /bin/sh.
//
// However, if a SHELL instruction is explicitly set, it indicates that using shell form is a conscious
// decision, and the warning should be suppressed.
//
// See: https://docs.docker.com/reference/build-checks/json-args-recommended/
func checkJSONArgsRecommended(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule

	// First pass: Check if a SHELL instruction is defined anywhere in the Dockerfile
	hasShellInstruction := false
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) == "SHELL" {
			hasShellInstruction = true
			break
		}
	}

	// Second pass: Check CMD and ENTRYPOINT instructions for shell form
	for _, child := range ast.Children {
		instruction := strings.ToUpper(child.Value)

		// Only check CMD and ENTRYPOINT
		if instruction != "CMD" && instruction != "ENTRYPOINT" {
			continue
		}

		// Extract the command from the original line
		trimmedOriginal := strings.TrimSpace(strings.TrimPrefix(child.Original, child.Value))

		// Skip if empty
		if trimmedOriginal == "" {
			continue
		}

		// Check if it's exec form (starts with '[')
		isExecForm := strings.HasPrefix(trimmedOriginal, "[")

		// Only flag shell form if no SHELL instruction is defined
		if !isExecForm && !hasShellInstruction {
			rules = append(rules, Rule{
				StartLine:   child.StartLine,
				EndLine:     child.EndLine,
				Code:        "JSONArgsRecommended",
				Description: "JSON arguments recommended for " + instruction + " to prevent unintended behavior related to OS signals",
				Url:         "https://docs.docker.com/reference/build-checks/json-args-recommended/",
				Severity:    SeverityWarning,
			})
		}
	}

	return rules
}

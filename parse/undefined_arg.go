package parse

import (
	"regexp"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkUndefinedArgInFrom checks that FROM instructions only reference ARGs that have been declared.
// ARG instructions before the first FROM are in global scope and can be used in FROM instructions.
// Docker also provides predefined ARGs like TARGETPLATFORM, BUILDPLATFORM, etc.
func checkUndefinedArgInFrom(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule
	globalArgs := make(map[string]bool)

	// Predefined ARGs that are automatically available
	// Source: https://docs.docker.com/reference/dockerfile/#automatic-platform-args-in-the-global-scope
	predefinedArgs := map[string]bool{
		"TARGETPLATFORM": true,
		"TARGETOS":       true,
		"TARGETARCH":     true,
		"TARGETVARIANT":  true,
		"BUILDPLATFORM":  true,
		"BUILDOS":        true,
		"BUILDARCH":      true,
		"BUILDVARIANT":   true,
		// HTTP proxy ARGs
		"HTTP_PROXY":  true,
		"http_proxy":  true,
		"HTTPS_PROXY": true,
		"https_proxy": true,
		"FTP_PROXY":   true,
		"ftp_proxy":   true,
		"NO_PROXY":    true,
		"no_proxy":    true,
		"ALL_PROXY":   true,
		"all_proxy":   true,
	}

	// First pass: collect global ARGs (before first FROM)
	for _, child := range ast.Children {
		insUppercase := strings.ToUpper(child.Value)

		if insUppercase == "FROM" {
			break
		}

		if insUppercase == "ARG" {
			// Extract ARG name(s) from the instruction
			argNames := extractArgNames(child)
			for _, argName := range argNames {
				globalArgs[argName] = true
			}
		}
	}

	// Second pass: check FROM instructions for undefined ARG references
	for _, child := range ast.Children {
		insUppercase := strings.ToUpper(child.Value)

		if insUppercase == "FROM" {
			// Extract the image reference from FROM
			imageRef, _, _ := extractFromComponents(child)

			// Find all variable references in the image reference
			varRefs := extractVariableReferences(imageRef)

			// Check each variable reference
			for _, varRef := range varRefs {
				// Check if it's defined in global ARGs or predefined ARGs
				if !globalArgs[varRef] && !predefinedArgs[varRef] {
					rules = append(rules, Rule{
						StartLine:   child.StartLine,
						EndLine:     child.EndLine,
						Code:        "UndefinedArgInFrom",
						Description: "FROM argument '" + varRef + "' is not declared",
						Url:         "https://docs.docker.com/reference/build-checks/undefined-arg-in-from/",
						Severity:    SeverityError,
					})
				}
			}
		}
	}

	return rules
}

// extractArgNames extracts all ARG names from an ARG instruction node
// ARG can have multiple arguments: ARG name1 name2=value
func extractArgNames(node *parser.Node) []string {
	var names []string

	current := node.Next
	for current != nil {
		argText := current.Value
		// ARG can be "NAME" or "NAME=value"
		// Extract just the name part
		if idx := strings.Index(argText, "="); idx != -1 {
			names = append(names, argText[:idx])
		} else {
			names = append(names, argText)
		}
		current = current.Next
	}

	return names
}

// extractVariableReferences finds all variable references in a string
// Matches both ${VAR} and $VAR formats
func extractVariableReferences(text string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Match ${VAR} format
	bracePattern := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	matches := bracePattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			refs = append(refs, match[1])
			seen[match[1]] = true
		}
	}

	// Match $VAR format (but not ${...} which we already handled)
	// The regex will match $VAR in both standalone "$VAR" and within "${VAR}"
	// We use the 'seen' map to deduplicate - if we already found it as ${VAR}, skip it
	dollarPattern := regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches = dollarPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			// Only add if we haven't seen this variable yet
			// (it may have been found as ${VAR} in the first pass)
			if !seen[varName] {
				refs = append(refs, varName)
				seen[varName] = true
			}
		}
	}

	return refs
}

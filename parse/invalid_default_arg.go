package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkInvalidDefaultArgInFrom validates that global ARG instructions used in
// FROM statements have appropriate default values.
//
// An ARG used in an image reference should be valid when no build arguments are
// provided. If an ARG without a default value is used in a FROM instruction and
// would result in an empty or invalid base image name, the build would fail unless
// the user provides --build-arg flags.
//
// Examples of invalid patterns:
//   - ARG TAG (without default) used in FROM busybox:${TAG} → results in busybox:
//   - ARG VERSION (without default) used in FROM image:${VERSION} → results in image:
//
// Valid patterns:
//   - ARG TAG=latest used in FROM busybox:${TAG}
//   - ARG VARIANT (without default) used in FROM busybox:stable${VARIANT} → results in busybox:stable
//   - ARG TAG used with fallback: FROM alpine:${TAG:-3.14}
func checkInvalidDefaultArgInFrom(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule

	// First pass: collect global ARGs (before first FROM) and their default values
	globalArgsWithDefaults := make(map[string]bool)
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) == "FROM" {
			break // Stop at first FROM
		}
		if strings.ToUpper(child.Value) == "ARG" {
			argNames := extractArgNamesWithDefaults(child)
			for argName, hasDefault := range argNames {
				globalArgsWithDefaults[argName] = hasDefault
			}
		}
	}

	// Second pass: check FROM instructions for ARG usage
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) != "FROM" {
			continue
		}

		imageRef, _, platformFlag := extractFromComponents(child)

		// Check image reference for variables
		imageVars := extractVariableReferences(imageRef)
		for _, varName := range imageVars {
			// Check if this is a global ARG without a default
			hasDefault, isGlobalArg := globalArgsWithDefaults[varName]
			if isGlobalArg && !hasDefault {
				// Check if the variable usage would result in invalid image reference
				if wouldResultInInvalidImageRef(imageRef, varName) {
					rules = append(rules, Rule{
						StartLine:   child.StartLine,
						EndLine:     child.EndLine,
						Code:        "InvalidDefaultArgInFrom",
						Description: "ARG '" + varName + "' has no default value and is used in FROM instruction. Provide a default value or use parameter expansion with fallback: ${" + varName + ":-default}",
						Url:         "https://docs.docker.com/reference/build-checks/invalid-default-arg-in-from/",
						Severity:    SeverityError,
					})
				}
			}
		}

		// Check platform flag for variables
		if platformFlag != "" {
			platformVars := extractVariableReferences(platformFlag)
			for _, varName := range platformVars {
				hasDefault, isGlobalArg := globalArgsWithDefaults[varName]
				if isGlobalArg && !hasDefault {
					// Platform flag with empty variable would be invalid
					rules = append(rules, Rule{
						StartLine:   child.StartLine,
						EndLine:     child.EndLine,
						Code:        "InvalidDefaultArgInFrom",
						Description: "ARG '" + varName + "' has no default value and is used in FROM --platform flag. Provide a default value or use parameter expansion with fallback: ${" + varName + ":-default}",
						Url:         "https://docs.docker.com/reference/build-checks/invalid-default-arg-in-from/",
						Severity:    SeverityError,
					})
				}
			}
		}
	}

	return rules
}

// extractArgNamesWithDefaults extracts ARG names and whether they have default values
func extractArgNamesWithDefaults(node *parser.Node) map[string]bool {
	result := make(map[string]bool)

	current := node.Next
	for current != nil {
		argValue := current.Value

		// Check if it has a default value (contains '=')
		if strings.Contains(argValue, "=") {
			parts := strings.SplitN(argValue, "=", 2)
			if len(parts) >= 1 {
				argName := parts[0]
				result[argName] = true // Has default
			}
		} else {
			// No default value
			result[argValue] = false
		}

		current = current.Next
	}

	return result
}

// wouldResultInInvalidImageRef checks if an empty variable would result in an invalid image reference
// Invalid patterns:
//   - image:${VAR} → image: (ends with colon)
//   - image@${VAR} → image@ (ends with @)
//   - ${VAR}/image → /image (starts with /)
//   - ${VAR}:tag → :tag (starts with colon)
//   - ${VAR}@digest → @digest (starts with @)
//   - foo/${VAR} → foo/ (ends with slash)
//   - foo/${VAR}/bar → foo//bar (double slash)
//
// Valid patterns:
//   - image${VAR} → image (still valid)
//   - image:stable${VAR} → image:stable (still valid)
//   - ${VAR}image → image (still valid if VAR is empty)
func wouldResultInInvalidImageRef(imageRef string, varName string) bool {
	// Check for ${VAR} or $VAR format
	patterns := []string{
		"${" + varName + "}",
		"$" + varName,
	}

	for _, pattern := range patterns {
		idx := strings.Index(imageRef, pattern)
		if idx == -1 {
			continue
		}

		// Check for parameter expansion with fallback (${VAR:-default})
		// This is valid even without a default ARG value
		if strings.Contains(pattern, "${"+varName) {
			// Look for :- or :+ syntax in the actual imageRef
			fullPattern := imageRef[idx:]
			if strings.Contains(fullPattern, ":-") || strings.Contains(fullPattern, ":+") {
				return false // Has fallback, so it's valid
			}
		}

		// Get character before and after the variable
		before := ""
		if idx > 0 {
			before = imageRef[idx-1 : idx]
		}

		after := ""
		endIdx := idx + len(pattern)
		if endIdx < len(imageRef) {
			after = imageRef[endIdx : endIdx+1]
		}

		// Check for invalid patterns:

		// 1. Variable immediately after : or @ (e.g., :${VAR} or @${VAR})
		// This results in trailing : or @ which is invalid
		if before == ":" || before == "@" {
			return true
		}

		// 2. Variable at start followed by : or @ (e.g., ${VAR}:tag or ${VAR}@digest)
		// This results in leading : or @ which is invalid
		if idx == 0 && (after == ":" || after == "@") {
			return true
		}

		// 3. Variable at the start followed by / (e.g., ${VAR}/image)
		// This results in /image which is invalid
		if idx == 0 && after == "/" {
			return true
		}

		// 4. Variable immediately after / (e.g., foo/${VAR})
		// If VAR is followed by nothing or /, this creates trailing or double slash
		if before == "/" {
			if after == "" || after == "/" || after == ":" || after == "@" {
				return true
			}
		}
	}

	return false
}

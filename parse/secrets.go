package parse

import (
	"regexp"
	"strings"
	"sync"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

var (
	secretsRegexp      *regexp.Regexp
	secretsAllowRegexp *regexp.Regexp
	secretsRegexpOnce  sync.Once
)

// getSecretsRegex returns compiled regex patterns for detecting secrets in variable names.
// The deny pattern matches common secret-related tokens (case insensitive) at word boundaries.
// The allow pattern matches tokens that should be excluded from secret detection.
func getSecretsRegex() (*regexp.Regexp, *regexp.Regexp) {
	// Check for either full value or first/last word.
	// Examples: api_key, DATABASE_PASSWORD, GITHUB_TOKEN, secret_MESSAGE, AUTH
	// Case insensitive.
	secretsRegexpOnce.Do(func() {
		secretTokens := []string{
			"apikey",
			"auth",
			"credential",
			"credentials",
			"key",
			"password",
			"pword",
			"passwd",
			"secret",
			"token",
		}
		pattern := `(?i)(?:_|^)(?:` + strings.Join(secretTokens, "|") + `)(?:_|$)`
		secretsRegexp = regexp.MustCompile(pattern)

		allowTokens := []string{
			"public",
		}
		allowPattern := `(?i)(?:_|^)(?:` + strings.Join(allowTokens, "|") + `)(?:_|$)`
		secretsAllowRegexp = regexp.MustCompile(allowPattern)
	})
	return secretsRegexp, secretsAllowRegexp
}

// checkSecretsInArgOrEnv validates that sensitive data is not exposed through
// ARG or ENV instructions in Dockerfiles.
//
// Secrets set in ARG or ENV instructions persist in the final image layers and
// metadata, potentially exposing them to anyone with access to the image.
// Docker recommends using secret mounts instead, which securely expose secrets
// during builds without persisting them in the final image.
//
// This check detects variable names that suggest sensitive information, such as:
// - SECRET, PASSWORD, TOKEN, KEY
// - AWS credentials (AWS_SECRET_ACCESS_KEY, AWS_ACCESS_KEY_ID)
// - Common patterns (API_KEY, PRIVATE_KEY, AUTH_TOKEN, etc.)
func checkSecretsInArgOrEnv(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule

	for _, child := range ast.Children {
		instruction := strings.ToUpper(child.Value)

		// Check ARG and ENV instructions
		if instruction == "ARG" || instruction == "ENV" {
			// Extract variable names
			varNames := extractVariableNamesFromInstruction(child)

			for _, varName := range varNames {
				if isSensitiveVariableName(varName) {
					rules = append(rules, Rule{
						StartLine:   child.StartLine,
						EndLine:     child.EndLine,
						Code:        "SecretsUsedInArgOrEnv",
						Description: "Sensitive data should not be used in " + instruction + " instruction: '" + varName + "'. Consider using secret mounts instead",
						Url:         "https://docs.docker.com/reference/build-checks/secrets-used-in-arg-or-env/",
						Severity:    SeverityWarning,
					})
				}
			}
		}
	}

	return rules
}

// extractVariableNamesFromInstruction extracts variable names from ARG or ENV instruction
func extractVariableNamesFromInstruction(node *parser.Node) []string {
	var names []string

	current := node.Next
	for current != nil {
		value := current.Value

		// Handle both "KEY=value" and "KEY value" formats
		if strings.Contains(value, "=") {
			// Extract key from "KEY=value"
			parts := strings.SplitN(value, "=", 2)
			if len(parts) >= 1 && parts[0] != "" {
				names = append(names, parts[0])
			}
		} else {
			// For space-separated format or just key name
			names = append(names, value)
		}

		current = current.Next
	}

	return names
}

// isSensitiveVariableName checks if a variable name suggests sensitive data
// using regex patterns that match common secret-related tokens at word boundaries.
// Variables containing allowlisted tokens (like "public") are excluded.
func isSensitiveVariableName(varName string) bool {
	deny, allow := getSecretsRegex()
	return deny.MatchString(varName) && !allow.MatchString(varName)
}

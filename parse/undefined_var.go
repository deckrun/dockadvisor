package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkUndefinedVar validates that all variable references are declared before use.
// This check applies to all instructions except shell form RUN, CMD, and ENTRYPOINT,
// where variables are resolved by the command shell at runtime.
//
// The check tracks variable scope per stage in multi-stage builds:
// - Global ARGs (before first FROM) are available to all stages
// - ARGs and ENVs within a stage are only available within that stage
// - Each FROM instruction starts a new stage with a fresh scope
//
// Predefined ARGs are automatically available:
// - Platform: TARGETPLATFORM, TARGETOS, TARGETARCH, TARGETVARIANT
// - Build: BUILDPLATFORM, BUILDOS, BUILDARCH, BUILDVARIANT
// - Proxy: HTTP_PROXY, HTTPS_PROXY, FTP_PROXY, NO_PROXY, ALL_PROXY (case-insensitive)
func checkUndefinedVar(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule

	// Predefined ARGs that are automatically available
	predefinedArgs := map[string]bool{
		"TARGETPLATFORM": true,
		"TARGETOS":       true,
		"TARGETARCH":     true,
		"TARGETVARIANT":  true,
		"BUILDPLATFORM":  true,
		"BUILDOS":        true,
		"BUILDARCH":      true,
		"BUILDVARIANT":   true,
		"HTTP_PROXY":     true,
		"http_proxy":     true,
		"HTTPS_PROXY":    true,
		"https_proxy":    true,
		"FTP_PROXY":      true,
		"ftp_proxy":      true,
		"NO_PROXY":       true,
		"no_proxy":       true,
		"ALL_PROXY":      true,
		"all_proxy":      true,
	}

	// Collect global ARGs (before first FROM)
	globalArgs := make(map[string]bool)
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) == "FROM" {
			break
		}
		if strings.ToUpper(child.Value) == "ARG" {
			argNames := extractArgNames(child)
			for _, argName := range argNames {
				globalArgs[argName] = true
			}
		}
	}

	// Process each stage
	inStage := false
	stageVars := make(map[string]bool) // Variables available in current stage

	for _, child := range ast.Children {
		instruction := strings.ToUpper(child.Value)

		// Start of a new stage
		if instruction == "FROM" {
			// Reset stage scope with global ARGs and predefined ARGs
			stageVars = make(map[string]bool)
			for k := range globalArgs {
				stageVars[k] = true
			}
			for k := range predefinedArgs {
				stageVars[k] = true
			}
			inStage = true

			// Check variables in FROM instruction (image reference and platform flag)
			imageRef, _, platformFlag := extractFromComponents(child)

			// Check image reference
			varRefs := extractVariableReferences(imageRef)
			for _, varName := range varRefs {
				if !stageVars[varName] && !predefinedArgs[varName] {
					rules = append(rules, Rule{
						StartLine:   child.StartLine,
						EndLine:     child.EndLine,
						Code:        "UndefinedVar",
						Description: "Usage of undefined variable '$" + varName + "'",
						Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
						Severity:    SeverityError,
					})
				}
			}

			// Check platform flag
			if platformFlag != "" {
				varRefs := extractVariableReferences(platformFlag)
				for _, varName := range varRefs {
					if !stageVars[varName] && !predefinedArgs[varName] {
						rules = append(rules, Rule{
							StartLine:   child.StartLine,
							EndLine:     child.EndLine,
							Code:        "UndefinedVar",
							Description: "Usage of undefined variable '$" + varName + "'",
							Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
						})
					}
				}
			}

			continue
		}

		if !inStage {
			continue // Skip instructions before first FROM
		}

		// Track ARG and ENV declarations in current stage
		if instruction == "ARG" {
			argNames := extractArgNames(child)
			for _, argName := range argNames {
				stageVars[argName] = true
			}
			// Also check if ARG uses undefined variables in default value
			if child.Next != nil {
				argValue := child.Next.Value
				// ARG can be "NAME" or "NAME=value"
				if strings.Contains(argValue, "=") {
					parts := strings.SplitN(argValue, "=", 2)
					if len(parts) == 2 {
						defaultValue := parts[1]
						varRefs := extractVariableReferences(defaultValue)
						for _, varName := range varRefs {
							if !stageVars[varName] && !predefinedArgs[varName] {
								rules = append(rules, Rule{
									StartLine:   child.StartLine,
									EndLine:     child.EndLine,
									Code:        "UndefinedVar",
									Description: "Usage of undefined variable '$" + varName + "'",
									Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
								})
							}
						}
					}
				}
			}
			continue
		}

		if instruction == "ENV" {
			envNames := extractEnvNames(child)
			for _, envName := range envNames {
				stageVars[envName] = true
			}
			// Check if ENV value uses undefined variables
			current := child.Next
			for current != nil {
				envValue := current.Value
				// ENV can be "KEY=value" or "KEY value"
				var valueToCheck string
				if strings.Contains(envValue, "=") {
					parts := strings.SplitN(envValue, "=", 2)
					if len(parts) == 2 {
						valueToCheck = parts[1]
					}
				} else {
					// Format: ENV KEY value (next node is the value)
					if current.Next != nil {
						valueToCheck = current.Next.Value
						current = current.Next // Skip the value node
					}
				}

				if valueToCheck != "" {
					varRefs := extractVariableReferences(valueToCheck)
					for _, varName := range varRefs {
						if !stageVars[varName] && !predefinedArgs[varName] {
							rules = append(rules, Rule{
								StartLine:   child.StartLine,
								EndLine:     child.EndLine,
								Code:        "UndefinedVar",
								Description: "Usage of undefined variable '$" + varName + "'",
								Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
							})
						}
					}
				}

				current = current.Next
			}
			continue
		}

		// Skip shell form RUN, CMD, and ENTRYPOINT (variables resolved by shell)
		if instruction == "RUN" || instruction == "CMD" || instruction == "ENTRYPOINT" {
			// Check if it's exec form (JSON array) or shell form
			// Exec form starts with [ in the original line
			if child.Attributes != nil && len(child.Attributes) > 0 {
				// Check if it's JSON form by looking at attributes
				isExecForm := false
				for key := range child.Attributes {
					if key == "json" {
						isExecForm = true
						break
					}
				}

				// For exec form, check for undefined variables
				if isExecForm {
					current := child.Next
					for current != nil {
						varRefs := extractVariableReferences(current.Value)
						for _, varName := range varRefs {
							if !stageVars[varName] && !predefinedArgs[varName] {
								rules = append(rules, Rule{
									StartLine:   child.StartLine,
									EndLine:     child.EndLine,
									Code:        "UndefinedVar",
									Description: "Usage of undefined variable '$" + varName + "'",
									Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
								})
							}
						}
						current = current.Next
					}
				}
				// Skip shell form (variables resolved by shell at runtime)
			}
			continue
		}

		// For all other instructions, check for undefined variables
		current := child.Next
		for current != nil {
			varRefs := extractVariableReferences(current.Value)
			for _, varName := range varRefs {
				if !stageVars[varName] && !predefinedArgs[varName] {
					rules = append(rules, Rule{
						StartLine:   child.StartLine,
						EndLine:     child.EndLine,
						Code:        "UndefinedVar",
						Description: "Usage of undefined variable '$" + varName + "'",
						Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
						Severity:    SeverityError,
					})
				}
			}

			// Check flags for variables (e.g., COPY --chown=$USER:$GROUP)
			for _, flag := range current.Flags {
				varRefs := extractVariableReferences(flag)
				for _, varName := range varRefs {
					if !stageVars[varName] && !predefinedArgs[varName] {
						rules = append(rules, Rule{
							StartLine:   child.StartLine,
							EndLine:     child.EndLine,
							Code:        "UndefinedVar",
							Description: "Usage of undefined variable '$" + varName + "'",
							Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
						})
					}
				}
			}

			current = current.Next
		}

		// Check node flags (e.g., FROM --platform=$VAR)
		for _, flag := range child.Flags {
			varRefs := extractVariableReferences(flag)
			for _, varName := range varRefs {
				if !stageVars[varName] && !predefinedArgs[varName] {
					rules = append(rules, Rule{
						StartLine:   child.StartLine,
						EndLine:     child.EndLine,
						Code:        "UndefinedVar",
						Description: "Usage of undefined variable '$" + varName + "'",
						Url:         "https://docs.docker.com/reference/build-checks/undefined-var/",
						Severity:    SeverityError,
					})
				}
			}
		}
	}

	return rules
}

// extractEnvNames extracts environment variable names from an ENV instruction node.
// ENV format can be:
//   - ENV KEY=value
//   - ENV KEY value
//   - ENV KEY1=value1 KEY2=value2 (multiple key-value pairs)
func extractEnvNames(node *parser.Node) []string {
	var names []string

	current := node.Next
	for current != nil {
		envValue := current.Value

		// Check if it's key=value format
		if strings.Contains(envValue, "=") {
			parts := strings.SplitN(envValue, "=", 2)
			if len(parts) >= 1 {
				names = append(names, parts[0])
			}
		} else {
			// Format: ENV KEY value (current is key, next is value)
			names = append(names, envValue)
			if current.Next != nil {
				current = current.Next // Skip the value
			}
		}

		current = current.Next
	}

	return names
}

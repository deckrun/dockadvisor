package parse

import (
	"encoding/json"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseRun(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "RUN requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Extract command and validate flags
	command := extractRunCommand(node)

	// Validate that command is not empty
	if strings.TrimSpace(command) == "" {
		return []Rule{NewErrorRule(node, "RunMissingCommand",
			"RUN instruction must specify a command to execute",
			"https://docs.docker.com/reference/dockerfile/#run")}
	}

	// Check if it's exec form (starts with '[')
	if strings.HasPrefix(strings.TrimSpace(command), "[") {
		if !checkExecFormJSON(command) {
			return []Rule{NewErrorRule(node, "RunInvalidExecForm",
				"RUN exec form must be a valid JSON array with double quotes",
				"https://docs.docker.com/reference/dockerfile/#run")}
		}
	}

	// Validate flags
	for _, flag := range node.Flags {
		// Validate --mount flag
		if strings.HasPrefix(flag, "--mount=") {
			mountValue := strings.TrimPrefix(flag, "--mount=")
			if !checkMountFlag(mountValue) {
				return []Rule{NewErrorRule(node, "RunInvalidMountFlag",
					"RUN --mount flag has invalid format: '"+flag+"'",
					"https://docs.docker.com/reference/dockerfile/#run---mount")}
			}
		}

		// Validate --network flag
		if strings.HasPrefix(flag, "--network=") {
			networkValue := strings.TrimPrefix(flag, "--network=")
			if !checkNetworkFlag(networkValue) {
				return []Rule{NewErrorRule(node, "RunInvalidNetworkFlag",
					"RUN --network flag must be one of: default, none, host. Got: '"+networkValue+"'",
					"https://docs.docker.com/reference/dockerfile/#run---network")}
			}
		}

		// Validate --security flag
		if strings.HasPrefix(flag, "--security=") {
			securityValue := strings.TrimPrefix(flag, "--security=")
			if !checkSecurityFlag(securityValue) {
				return []Rule{NewErrorRule(node, "RunInvalidSecurityFlag",
					"RUN --security flag must be one of: sandbox, insecure. Got: '"+securityValue+"'",
					"https://docs.docker.com/reference/dockerfile/#run---security")}
			}
		}
	}

	// No warnings in this file
	return nil
}

// extractRunCommand extracts the command string from the RUN instruction
func extractRunCommand(node *parser.Node) string {
	if node.Next == nil {
		return ""
	}

	// Collect all arguments into a single command string
	var parts []string
	current := node.Next
	for current != nil {
		parts = append(parts, current.Value)
		current = current.Next
	}

	return strings.Join(parts, " ")
}

// checkExecFormJSON validates that the exec form is valid JSON array
func checkExecFormJSON(command string) bool {
	command = strings.TrimSpace(command)

	// Must start with '[' and end with ']'
	if !strings.HasPrefix(command, "[") || !strings.HasSuffix(command, "]") {
		return false
	}

	// Try to parse as JSON array
	var arr []string
	if err := json.Unmarshal([]byte(command), &arr); err != nil {
		return false
	}

	// Array should not be empty
	if len(arr) == 0 {
		return false
	}

	// Check that original string uses double quotes, not single quotes
	// Single quotes are not valid JSON
	if strings.Contains(command, "['") || strings.Contains(command, "']") {
		return false
	}

	return true
}

// checkMountFlag validates the --mount flag format
func checkMountFlag(mountValue string) bool {
	if mountValue == "" {
		return false
	}

	// Parse mount options (format: type=<type>[,option=value,...]
	// Valid types: bind, cache, tmpfs, secret, ssh
	if strings.HasPrefix(mountValue, "type=") {
		parts := strings.SplitN(mountValue, ",", 2)
		typeValue := strings.TrimPrefix(parts[0], "type=")

		validTypes := map[string]bool{
			"bind":   true,
			"cache":  true,
			"tmpfs":  true,
			"secret": true,
			"ssh":    true,
		}

		return validTypes[typeValue]
	}

	// If no type specified, it defaults to bind, which is valid
	// Just check that it has some content
	return len(mountValue) > 0
}

// checkNetworkFlag validates the --network flag value
func checkNetworkFlag(networkValue string) bool {
	validNetworks := map[string]bool{
		"default": true,
		"none":    true,
		"host":    true,
	}

	return validNetworks[networkValue]
}

// checkSecurityFlag validates the --security flag value
func checkSecurityFlag(securityValue string) bool {
	validSecurity := map[string]bool{
		"sandbox":  true,
		"insecure": true,
	}

	return validSecurity[securityValue]
}

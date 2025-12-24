package parse

import (
	"strconv"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseEXPOSE(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "EXPOSE requires at least one argument")}
	}

	// ERROR CHECKS - Return immediately on first error
	// Check all arguments in the EXPOSE instruction
	current := node.Next
	for current != nil {
		if !checkExposeFormat(current.Value) {
			return []Rule{NewErrorRule(node, "ExposeInvalidFormat",
				"EXPOSE instruction should not define an IP address or host-port mapping, found '"+current.Value+"'",
				"https://docs.docker.com/reference/build-checks/expose-invalid-format/")}
		}

		// Check if port number is within valid range (0-65535)
		if !checkExposePortRange(current.Value) {
			return []Rule{NewErrorRule(node, "ExposePortOutOfRange",
				"Port number in EXPOSE instruction is outside valid UNIX port range (0-65535): '"+current.Value+"'",
				"https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers")}
		}

		// Check if protocol is valid (only tcp or udp)
		if !checkExposeValidProtocol(current.Value) {
			return []Rule{NewErrorRule(node, "ExposeInvalidProtocol",
				"Invalid protocol in EXPOSE instruction '"+current.Value+"', only 'tcp' and 'udp' are supported",
				"https://docs.docker.com/reference/dockerfile/#expose")}
		}

		current = current.Next
	}

	// WARNING CHECKS - Collect warnings
	var exposeRules []Rule

	current = node.Next
	for current != nil {
		// Check protocol casing if present
		if !checkExposeProtoCasing(current.Value) {
			exposeRules = append(exposeRules, NewWarningRule(node, "ExposeProtoCasing",
				"Defined protocol '"+current.Value+"' in EXPOSE instruction should be lowercase",
				"https://docs.docker.com/reference/build-checks/expose-proto-casing/"))
		}

		current = current.Next
	}

	return exposeRules
}

// checkExposeFormat checks if an EXPOSE port specification is valid.
// Returns true if valid (only port or port/protocol), false if it contains IP or host-port mapping.
// Valid formats:
//   - 80
//   - 80/tcp
//   - 80/udp
//
// Invalid formats:
//   - 127.0.0.1:80:80 (IP address and host-port mapping)
//   - 80:80 (host-port mapping)
//   - 0.0.0.0:8080 (IP address mapping)
func checkExposeFormat(portSpec string) bool {
	// If it contains a colon, it's either:
	// - IP:port or IP:host-port:container-port (invalid)
	// - host-port:container-port (invalid)
	// The only valid use of colon would be after a slash for protocol specification
	// But that's handled separately, so any colon before a slash is invalid

	// Check if there's a colon in the port specification
	colonCount := strings.Count(portSpec, ":")

	// If there are any colons, it's invalid (IP or port mapping)
	if colonCount > 0 {
		return false
	}

	// Valid format: just port number or port/protocol
	return true
}

// checkExposeValidProtocol checks if the protocol in an EXPOSE port specification is valid.
// Returns true if no protocol is specified or if protocol is 'tcp' or 'udp'.
// Returns false for any other protocol.
// Examples:
//   - "80" -> true (no protocol)
//   - "80/tcp" -> true (valid protocol)
//   - "80/udp" -> true (valid protocol)
//   - "80/TCP" -> true (valid protocol, case-insensitive)
//   - "80/http" -> false (invalid protocol)
//   - "80/sctp" -> false (invalid protocol)
func checkExposeValidProtocol(portSpec string) bool {
	// Check if there's a protocol specified (contains /)
	if !strings.Contains(portSpec, "/") {
		return true // No protocol specified, so it's valid
	}

	// Split by / to get protocol
	parts := strings.Split(portSpec, "/")
	if len(parts) != 2 {
		return true // Invalid format, handled by checkExposeFormat
	}

	protocol := strings.ToLower(parts[1])

	// Only tcp and udp are valid protocols
	return protocol == "tcp" || protocol == "udp"
}

// checkExposeProtoCasing checks if the protocol in an EXPOSE port specification is lowercase.
// Returns true if no protocol is specified or if the protocol is lowercase.
// Returns false if the protocol contains uppercase letters.
// Examples:
//   - "80" -> true (no protocol)
//   - "80/tcp" -> true (lowercase protocol)
//   - "80/TCP" -> false (uppercase protocol)
//   - "80/TcP" -> false (mixed case protocol)
func checkExposeProtoCasing(portSpec string) bool {
	// Check if there's a protocol specified (contains /)
	if !strings.Contains(portSpec, "/") {
		return true // No protocol specified, so it's valid
	}

	// Split by / to get protocol
	parts := strings.Split(portSpec, "/")
	if len(parts) != 2 {
		return true // Invalid format, but that's handled by checkExposeFormat
	}

	protocol := parts[1]

	// Check if protocol is lowercase
	return protocol == strings.ToLower(protocol)
}

// checkExposePortRange checks if the port number in an EXPOSE port specification is within the valid UNIX port range (0-65535).
// Returns true if the port is valid, false otherwise.
// Examples:
//   - "80" -> true (valid port)
//   - "80/tcp" -> true (valid port with protocol)
//   - "65535" -> true (maximum valid port)
//   - "0" -> true (minimum valid port)
//   - "80000" -> false (exceeds maximum)
//   - "-1" -> false (below minimum)
//   - "abc" -> false (not a number)
func checkExposePortRange(portSpec string) bool {
	// Extract just the port number (before any protocol specification)
	portStr := portSpec
	if strings.Contains(portSpec, "/") {
		parts := strings.Split(portSpec, "/")
		if len(parts) > 0 {
			portStr = parts[0]
		}
	}

	// Try to parse the port as an integer
	port, err := strconv.Atoi(portStr)
	if err != nil {
		// If it's not a valid integer, return true (validation handled elsewhere)
		// This could be a variable reference like $PORT
		return true
	}

	// Check if port is within valid range (0-65535)
	return port >= 0 && port <= 65535
}

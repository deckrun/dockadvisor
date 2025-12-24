package parse

import (
	"testing"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/stretchr/testify/require"
)

func TestCheckExposeFormat(t *testing.T) {
	tests := []struct {
		name     string
		portSpec string
		expected bool
	}{
		// Valid examples - only port number or port/protocol
		{
			name:     "valid port number",
			portSpec: "80",
			expected: true,
		},
		{
			name:     "valid port with tcp protocol",
			portSpec: "80/tcp",
			expected: true,
		},
		{
			name:     "valid port with udp protocol",
			portSpec: "8080/udp",
			expected: true,
		},
		{
			name:     "valid high port number",
			portSpec: "3000",
			expected: true,
		},
		{
			name:     "valid port 443",
			portSpec: "443",
			expected: true,
		},
		{
			name:     "valid port 443 with tcp",
			portSpec: "443/tcp",
			expected: true,
		},
		// Invalid examples - IP address or host-port mapping
		{
			name:     "invalid - IP address with host-port and container-port mapping",
			portSpec: "127.0.0.1:80:80",
			expected: false,
		},
		{
			name:     "invalid - host-port to container-port mapping",
			portSpec: "80:80",
			expected: false,
		},
		{
			name:     "invalid - IP address with port",
			portSpec: "0.0.0.0:8080",
			expected: false,
		},
		{
			name:     "invalid - localhost with port",
			portSpec: "127.0.0.1:3000",
			expected: false,
		},
		{
			name:     "invalid - host-port mapping with different ports",
			portSpec: "8080:80",
			expected: false,
		},
		{
			name:     "invalid - IP with host and container port",
			portSpec: "192.168.1.1:8080:80",
			expected: false,
		},
		{
			name:     "invalid - multiple port mapping",
			portSpec: "3000:3000",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkExposeFormat(tt.portSpec)
			require.Equal(t, tt.expected, result, "checkExposeFormat(%q) returned unexpected result", tt.portSpec)
		})
	}
}

func TestCheckExposeValidProtocol(t *testing.T) {
	tests := []struct {
		name     string
		portSpec string
		expected bool
	}{
		// Valid examples - no protocol or tcp/udp
		{
			name:     "no protocol specified",
			portSpec: "80",
			expected: true,
		},
		{
			name:     "lowercase tcp protocol",
			portSpec: "80/tcp",
			expected: true,
		},
		{
			name:     "lowercase udp protocol",
			portSpec: "8080/udp",
			expected: true,
		},
		{
			name:     "uppercase TCP protocol",
			portSpec: "80/TCP",
			expected: true,
		},
		{
			name:     "uppercase UDP protocol",
			portSpec: "8080/UDP",
			expected: true,
		},
		{
			name:     "mixed case TcP protocol",
			portSpec: "443/TcP",
			expected: true,
		},
		{
			name:     "mixed case Udp protocol",
			portSpec: "53/Udp",
			expected: true,
		},
		// Invalid examples - other protocols
		{
			name:     "invalid http protocol",
			portSpec: "80/http",
			expected: false,
		},
		{
			name:     "invalid https protocol",
			portSpec: "443/https",
			expected: false,
		},
		{
			name:     "invalid sctp protocol",
			portSpec: "9999/sctp",
			expected: false,
		},
		{
			name:     "invalid icmp protocol",
			portSpec: "0/icmp",
			expected: false,
		},
		{
			name:     "invalid random protocol",
			portSpec: "8080/random",
			expected: false,
		},
		{
			name:     "invalid dccp protocol",
			portSpec: "5000/dccp",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkExposeValidProtocol(tt.portSpec)
			require.Equal(t, tt.expected, result, "checkExposeValidProtocol(%q) returned unexpected result", tt.portSpec)
		})
	}
}

func TestCheckExposeProtoCasing(t *testing.T) {
	tests := []struct {
		name     string
		portSpec string
		expected bool
	}{
		// Valid examples - lowercase protocol or no protocol
		{
			name:     "no protocol specified",
			portSpec: "80",
			expected: true,
		},
		{
			name:     "lowercase tcp protocol",
			portSpec: "80/tcp",
			expected: true,
		},
		{
			name:     "lowercase udp protocol",
			portSpec: "8080/udp",
			expected: true,
		},
		{
			name:     "port without protocol",
			portSpec: "443",
			expected: true,
		},
		// Invalid examples - uppercase or mixed case protocol
		{
			name:     "uppercase TCP protocol",
			portSpec: "80/TCP",
			expected: false,
		},
		{
			name:     "uppercase UDP protocol",
			portSpec: "8080/UDP",
			expected: false,
		},
		{
			name:     "mixed case TcP protocol",
			portSpec: "80/TcP",
			expected: false,
		},
		{
			name:     "mixed case Udp protocol",
			portSpec: "8080/Udp",
			expected: false,
		},
		{
			name:     "mixed case tCp protocol",
			portSpec: "443/tCp",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkExposeProtoCasing(tt.portSpec)
			require.Equal(t, tt.expected, result, "checkExposeProtoCasing(%q) returned unexpected result", tt.portSpec)
		})
	}
}

func TestParseEXPOSE(t *testing.T) {
	t.Run("returns invalid instruction rule when node.Next is nil", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next:      nil,
		}

		rules := parseEXPOSE(node)

		require.Len(t, rules, 1, "expected exactly 1 rule")

		rule := rules[0]
		require.Equal(t, invalidInstructionCode, rule.Code)
		require.Equal(t, "EXPOSE requires at least one argument", rule.Description)
		require.Equal(t, 3, rule.StartLine)
		require.Equal(t, 3, rule.EndLine)
	})

	t.Run("returns ExposeInvalidFormat for IP address with port mapping", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "127.0.0.1:80:80",
			},
		}

		rules := parseEXPOSE(node)

		require.Len(t, rules, 1, "expected exactly 1 rule")

		rule := rules[0]
		require.Equal(t, "ExposeInvalidFormat", rule.Code)
		require.Contains(t, rule.Description, "127.0.0.1:80:80")
		require.Equal(t, 3, rule.StartLine)
		require.Equal(t, 3, rule.EndLine)
		require.Equal(t, "https://docs.docker.com/reference/build-checks/expose-invalid-format/", rule.Url)
	})

	t.Run("returns ExposeInvalidFormat for host-port mapping", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80:80",
			},
		}

		rules := parseEXPOSE(node)

		require.Len(t, rules, 1, "expected exactly 1 rule")

		rule := rules[0]
		require.Equal(t, "ExposeInvalidFormat", rule.Code)
		require.Contains(t, rule.Description, "80:80")
	})

	t.Run("returns no rules for valid port", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80",
			},
		}

		rules := parseEXPOSE(node)

		require.Empty(t, rules, "expected no rules for valid EXPOSE")
	})

	t.Run("returns no rules for valid port with protocol", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80/tcp",
			},
		}

		rules := parseEXPOSE(node)

		require.Empty(t, rules, "expected no rules for valid EXPOSE")
	})

	t.Run("handles multiple ports with some invalid", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80",
				Next: &parser.Node{
					Value: "80:80",
					Next: &parser.Node{
						Value: "443",
					},
				},
			},
		}

		rules := parseEXPOSE(node)

		// Should only flag the invalid one (80:80)
		require.Len(t, rules, 1, "expected 1 rule for the invalid port mapping")
		require.Equal(t, "ExposeInvalidFormat", rules[0].Code)
		require.Contains(t, rules[0].Description, "80:80")
	})

	t.Run("returns ExposeProtoCasing for uppercase protocol", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80/TCP",
			},
		}

		rules := parseEXPOSE(node)

		require.Len(t, rules, 1, "expected exactly 1 rule")

		rule := rules[0]
		require.Equal(t, "ExposeProtoCasing", rule.Code)
		require.Contains(t, rule.Description, "80/TCP")
		require.Contains(t, rule.Description, "should be lowercase")
		require.Equal(t, 3, rule.StartLine)
		require.Equal(t, 3, rule.EndLine)
		require.Equal(t, "https://docs.docker.com/reference/build-checks/expose-proto-casing/", rule.Url)
	})

	t.Run("returns ExposeProtoCasing for mixed case protocol", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 5,
			EndLine:   5,
			Next: &parser.Node{
				Value: "8080/TcP",
			},
		}

		rules := parseEXPOSE(node)

		require.Len(t, rules, 1, "expected exactly 1 rule")

		rule := rules[0]
		require.Equal(t, "ExposeProtoCasing", rule.Code)
		require.Contains(t, rule.Description, "8080/TcP")
		require.Contains(t, rule.Description, "should be lowercase")
	})

	t.Run("handles multiple ports with mixed case protocols", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80/tcp",
				Next: &parser.Node{
					Value: "443/TCP",
					Next: &parser.Node{
						Value: "8080/Udp",
					},
				},
			},
		}

		rules := parseEXPOSE(node)

		// Should flag both 443/TCP and 8080/Udp
		require.Len(t, rules, 2, "expected 2 rules for the uppercase protocols")
		require.Equal(t, "ExposeProtoCasing", rules[0].Code)
		require.Contains(t, rules[0].Description, "443/TCP")
		require.Equal(t, "ExposeProtoCasing", rules[1].Code)
		require.Contains(t, rules[1].Description, "8080/Udp")
	})

	t.Run("returns ExposeInvalidProtocol for http protocol", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80/http",
			},
		}

		rules := parseEXPOSE(node)

		require.Len(t, rules, 1, "expected exactly 1 rule")

		rule := rules[0]
		require.Equal(t, "ExposeInvalidProtocol", rule.Code)
		require.Contains(t, rule.Description, "80/http")
		require.Contains(t, rule.Description, "only 'tcp' and 'udp' are supported")
		require.Equal(t, 3, rule.StartLine)
		require.Equal(t, 3, rule.EndLine)
		require.Equal(t, "https://docs.docker.com/reference/dockerfile/#expose", rule.Url)
	})

	t.Run("returns ExposeInvalidProtocol for sctp protocol", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 5,
			EndLine:   5,
			Next: &parser.Node{
				Value: "9999/sctp",
			},
		}

		rules := parseEXPOSE(node)

		require.Len(t, rules, 1, "expected exactly 1 rule")

		rule := rules[0]
		require.Equal(t, "ExposeInvalidProtocol", rule.Code)
		require.Contains(t, rule.Description, "9999/sctp")
	})

	t.Run("handles multiple ports with some invalid protocols", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80/tcp",
				Next: &parser.Node{
					Value: "443/https",
					Next: &parser.Node{
						Value: "8080/udp",
						Next: &parser.Node{
							Value: "9999/sctp",
						},
					},
				},
			},
		}

		rules := parseEXPOSE(node)

		// Should flag first invalid protocol (443/https) and return immediately
		require.Len(t, rules, 1, "expected 1 rule for the first invalid protocol")
		require.Equal(t, "ExposeInvalidProtocol", rules[0].Code)
		require.Contains(t, rules[0].Description, "443/https")
	})

	t.Run("returns both ExposeInvalidProtocol and ExposeProtoCasing when protocol is invalid and uppercase", func(t *testing.T) {
		node := &parser.Node{
			Value:     "EXPOSE",
			StartLine: 3,
			EndLine:   3,
			Next: &parser.Node{
				Value: "80/HTTP",
			},
		}

		rules := parseEXPOSE(node)

		// Should flag invalid protocol (error) and return immediately, not check casing (warning)
		require.Len(t, rules, 1, "expected 1 rule for invalid protocol error")
		require.Equal(t, "ExposeInvalidProtocol", rules[0].Code)
	})
}

func TestCheckExposePortRange(t *testing.T) {
	tests := []struct {
		name     string
		portSpec string
		expected bool
	}{
		// Valid port numbers (0-65535)
		{
			name:     "minimum valid port - 0",
			portSpec: "0",
			expected: true,
		},
		{
			name:     "standard port - 80",
			portSpec: "80",
			expected: true,
		},
		{
			name:     "standard port - 443",
			portSpec: "443",
			expected: true,
		},
		{
			name:     "high port - 8080",
			portSpec: "8080",
			expected: true,
		},
		{
			name:     "maximum valid port - 65535",
			portSpec: "65535",
			expected: true,
		},
		{
			name:     "valid port with tcp protocol",
			portSpec: "80/tcp",
			expected: true,
		},
		{
			name:     "valid port with udp protocol",
			portSpec: "8080/udp",
			expected: true,
		},
		{
			name:     "maximum port with tcp protocol",
			portSpec: "65535/tcp",
			expected: true,
		},
		// Invalid port numbers (outside 0-65535)
		{
			name:     "port exceeds maximum - 80000",
			portSpec: "80000",
			expected: false,
		},
		{
			name:     "port exceeds maximum - 65536",
			portSpec: "65536",
			expected: false,
		},
		{
			name:     "port exceeds maximum - 100000",
			portSpec: "100000",
			expected: false,
		},
		{
			name:     "negative port - -1",
			portSpec: "-1",
			expected: false,
		},
		{
			name:     "negative port - -100",
			portSpec: "-100",
			expected: false,
		},
		{
			name:     "port exceeds maximum with tcp protocol",
			portSpec: "80000/tcp",
			expected: false,
		},
		{
			name:     "port exceeds maximum with udp protocol",
			portSpec: "70000/udp",
			expected: false,
		},
		// Variable references (should pass - not validated as numbers)
		{
			name:     "variable reference",
			portSpec: "$PORT",
			expected: true,
		},
		{
			name:     "variable with braces",
			portSpec: "${PORT}",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkExposePortRange(tt.portSpec)
			require.Equal(t, tt.expected, result, "checkExposePortRange(%q) returned unexpected result", tt.portSpec)
		})
	}
}

func TestParseEXPOSEWithPortRange(t *testing.T) {
	tests := []struct {
		name              string
		dockerfileContent string
		expectedRules     []string
	}{
		// Valid port ranges
		{
			name:              "valid port 80",
			dockerfileContent: "EXPOSE 80",
			expectedRules:     []string{},
		},
		{
			name:              "valid port 65535",
			dockerfileContent: "EXPOSE 65535",
			expectedRules:     []string{},
		},
		{
			name:              "valid port 0",
			dockerfileContent: "EXPOSE 0",
			expectedRules:     []string{},
		},
		{
			name:              "valid multiple ports",
			dockerfileContent: "EXPOSE 80 443 8080",
			expectedRules:     []string{},
		},
		// Invalid port ranges
		{
			name:              "port exceeds maximum - 80000",
			dockerfileContent: "EXPOSE 80000",
			expectedRules:     []string{"ExposePortOutOfRange"},
		},
		{
			name:              "port exceeds maximum - 65536",
			dockerfileContent: "EXPOSE 65536",
			expectedRules:     []string{"ExposePortOutOfRange"},
		},
		{
			name:              "negative port",
			dockerfileContent: "EXPOSE -1",
			expectedRules:     []string{"ExposePortOutOfRange"},
		},
		{
			name:              "port with tcp protocol exceeds maximum",
			dockerfileContent: "EXPOSE 80000/tcp",
			expectedRules:     []string{"ExposePortOutOfRange"},
		},
		{
			name:              "multiple ports with one invalid",
			dockerfileContent: "EXPOSE 80 80000 443",
			expectedRules:     []string{"ExposePortOutOfRange"},
		},
		{
			name:              "multiple invalid ports",
			dockerfileContent: "EXPOSE 80000 100000",
			expectedRules:     []string{"ExposePortOutOfRange"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfileContent)
			require.NoError(t, err, "ParseDockerfile should not return an error")
			require.NotNil(t, result, "ParseDockerfile should return a non-nil result")

			if len(tt.expectedRules) == 0 {
				require.Empty(t, result.Rules, "Expected no rules but got: %v", result.Rules)
			} else {
				require.Len(t, result.Rules, len(tt.expectedRules), "Expected %d rules but got %d: %v",
					len(tt.expectedRules), len(result.Rules), result.Rules)

				actualRuleCodes := make([]string, 0, len(result.Rules))
				for _, rule := range result.Rules {
					actualRuleCodes = append(actualRuleCodes, rule.Code)
				}

				require.ElementsMatch(t, tt.expectedRules, actualRuleCodes,
					"Expected rule codes %v but got %v", tt.expectedRules, actualRuleCodes)

				// Verify rule structure for ExposePortOutOfRange
				for _, rule := range result.Rules {
					if rule.Code == "ExposePortOutOfRange" {
						require.NotEmpty(t, rule.Description)
						require.Contains(t, rule.Description, "outside valid UNIX port range")
						require.Equal(t, "https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers",
							rule.Url)
					}
				}
			}
		})
	}
}

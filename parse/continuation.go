package parse

import (
	"strings"
)

// checkEmptyContinuations scans the dockerfile content for empty continuation lines.
// Empty continuation lines are empty lines following a newline escape character (\).
// These are deprecated and will generate errors in future versions of Docker.
// Lines with comments are not considered empty.
func checkEmptyContinuations(dockerfileContent string) []Rule {
	var rules []Rule
	lines := strings.Split(dockerfileContent, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if this line ends with a backslash continuation
		trimmedLine := strings.TrimRight(line, " \t\r")
		if !strings.HasSuffix(trimmedLine, "\\") {
			continue
		}

		// Check if there's a next line
		if i+1 >= len(lines) {
			continue
		}

		nextLine := lines[i+1]

		// Check if the next line is empty (only whitespace, no comment)
		trimmedNextLine := strings.TrimSpace(nextLine)

		// If the next line is empty or only contains whitespace (no comment), it's a violation
		if trimmedNextLine == "" {
			// Find the instruction this continuation belongs to
			// Search backwards to find the start of the instruction
			instructionLine := i + 1 // Start from current line (1-indexed)
			for j := i; j >= 0; j-- {
				prevLine := strings.TrimSpace(lines[j])
				if prevLine != "" && !strings.HasSuffix(strings.TrimRight(lines[j], " \t\r"), "\\") {
					instructionLine = j + 1 // Convert to 1-indexed
					break
				}
			}

			rules = append(rules, Rule{
				StartLine:   instructionLine,
				EndLine:     i + 2, // The empty line is at i+1, convert to 1-indexed
				Code:        "NoEmptyContinuation",
				Description: "Empty continuation line found. Empty lines following a backslash continuation are deprecated and will cause errors in future Docker versions.",
				Url:         "https://docs.docker.com/reference/build-checks/no-empty-continuation/",
				Severity:    SeverityWarning,
			})

			// Skip the empty line to avoid duplicate reports
			i++
		}
	}

	return rules
}

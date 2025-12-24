package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func parseWorkdir(node *parser.Node) []Rule {
	if node.Next == nil {
		return []Rule{invalidInstructionRule(node, "WORKDIR requires exactly one argument")}
	}

	workdirValue := node.Next
	var workdirRules []Rule

	if !checkWorkdirAbsolute(workdirValue.Value) {
		workdirRules = append(workdirRules, NewWarningRule(node, "WorkdirRelativePath",
			"WORKDIR uses a relative path. Consider using an absolute path (starting with /) to avoid issues when the base image's working directory changes.",
			"https://docs.docker.com/reference/build-checks/workdir-relative-path/"))
	}

	return workdirRules
}

// checkWorkdirAbsolute checks if a WORKDIR path is absolute.
// Returns true if the path is absolute, false if it's relative.
// Absolute paths include:
//   - Unix absolute paths (starting with /)
//   - Windows absolute paths (e.g., C:, C:\, C:/)
//   - Variable references (e.g., $HOME, ${WORKDIR}) - considered "absolute enough"
func checkWorkdirAbsolute(workdirPath string) bool {
	// Unix absolute path
	if strings.HasPrefix(workdirPath, "/") {
		return true
	}

	// Windows absolute path (C:, C:\, C:/)
	// Check for drive letter followed by colon
	if len(workdirPath) >= 2 && workdirPath[1] == ':' {
		// First character should be a letter (drive letter)
		firstChar := workdirPath[0]
		if (firstChar >= 'A' && firstChar <= 'Z') || (firstChar >= 'a' && firstChar <= 'z') {
			return true
		}
	}

	// Variable reference (considered "absolute enough")
	// Variables like $HOME, ${WORKDIR}, $APP_HOME are treated as absolute
	// since they will be resolved at build time and we can't know if they're absolute
	if strings.HasPrefix(workdirPath, "$") {
		return true
	}

	return false
}

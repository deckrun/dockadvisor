package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkDuplicateStageNames checks that all stage names in the Dockerfile are unique.
// Stage names are compared case-insensitively since Docker treats stage names in a case-insensitive manner.
// Returns rules for any duplicate stage name declarations.
func checkDuplicateStageNames(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule
	stageNames := make(map[string][]stageInfo)

	// Collect all stage names from FROM instructions
	for _, child := range ast.Children {
		if strings.ToUpper(child.Value) != "FROM" {
			continue
		}

		_, stageName, _ := extractFromComponents(child)
		if stageName == "" {
			continue
		}

		// Normalize to lowercase for case-insensitive comparison
		lowerStageName := strings.ToLower(stageName)

		stageNames[lowerStageName] = append(stageNames[lowerStageName], stageInfo{
			originalName: stageName,
			startLine:    child.StartLine,
			endLine:      child.EndLine,
		})
	}

	// Find duplicates
	for _, stages := range stageNames {
		if len(stages) > 1 {
			// Report all occurrences of the duplicate
			for _, stage := range stages {
				rules = append(rules, Rule{
					StartLine:   stage.startLine,
					EndLine:     stage.endLine,
					Code:        "DuplicateStageName",
					Description: "Duplicate stage name '" + stage.originalName + "', stage names should be unique",
					Url:         "https://docs.docker.com/reference/build-checks/duplicate-stage-name/",
					Severity:    SeverityError,
				})
			}
		}
	}

	return rules
}

type stageInfo struct {
	originalName string
	startLine    int
	endLine      int
}

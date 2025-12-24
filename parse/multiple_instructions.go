package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkMultipleInstructionsDisallowed validates that certain instructions
// (CMD, HEALTHCHECK, ENTRYPOINT) appear at most once per stage.
//
// According to Docker best practices, an image can only have one CMD, HEALTHCHECK,
// and ENTRYPOINT. When multiple instructions of the same type are present, only the
// last occurrence is used, and earlier ones are silently ignored, which can lead to
// confusion and unintended behavior.
//
// This check flags all occurrences after the first one within each stage.
// In multi-stage builds, each stage can have its own CMD/HEALTHCHECK/ENTRYPOINT.
func checkMultipleInstructionsDisallowed(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule

	// Instructions that should appear at most once per stage
	restrictedInstructions := map[string]bool{
		"CMD":         true,
		"HEALTHCHECK": true,
		"ENTRYPOINT":  true,
	}

	// Track occurrences per stage
	type instructionInfo struct {
		count     int
		firstLine int
		endLine   int
	}

	// Track per stage
	stageInstructions := make(map[string]*instructionInfo)
	inStage := false

	for _, child := range ast.Children {
		instruction := strings.ToUpper(child.Value)

		// Start of a new stage
		if instruction == "FROM" {
			// Reset stage tracking
			stageInstructions = make(map[string]*instructionInfo)
			inStage = true
			continue
		}

		if !inStage {
			continue // Skip instructions before first FROM
		}

		// Check if this is a restricted instruction
		if restrictedInstructions[instruction] {
			if info, exists := stageInstructions[instruction]; exists {
				// This is not the first occurrence
				info.count++
				rules = append(rules, Rule{
					StartLine:   child.StartLine,
					EndLine:     child.EndLine,
					Code:        "MultipleInstructionsDisallowed",
					Description: "Multiple " + instruction + " instructions should not be used in the same stage; only the last one takes effect",
					Url:         "https://docs.docker.com/reference/build-checks/multiple-instructions-disallowed/",
					Severity:    SeverityError,
				})
			} else {
				// First occurrence
				stageInstructions[instruction] = &instructionInfo{
					count:     1,
					firstLine: child.StartLine,
					endLine:   child.EndLine,
				}
			}
		}
	}

	return rules
}

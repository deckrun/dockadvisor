package parse

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// checkConsistentInstructionCasing checks that all instruction keywords use consistent casing.
// Instructions should be either all uppercase or all lowercase, not mixed.
// Returns rules for instructions that don't match the predominant casing style.
func checkConsistentInstructionCasing(ast *parser.Node) []Rule {
	if ast == nil || len(ast.Children) == 0 {
		return nil
	}

	var rules []Rule

	// Count uppercase vs lowercase instructions
	var uppercaseCount, lowercaseCount int
	for _, child := range ast.Children {
		instruction := child.Value
		if instruction == "" {
			continue
		}

		if instruction == strings.ToUpper(instruction) {
			uppercaseCount++
		} else if instruction == strings.ToLower(instruction) {
			lowercaseCount++
		}
		// Mixed case instructions (e.g., "From", "Run") are neither pure upper nor pure lower
	}

	// Determine the predominant style
	// If all instructions are the same case, no violations
	if uppercaseCount == 0 || lowercaseCount == 0 {
		// Check for mixed-case instructions when all others are consistent
		expectedCase := "uppercase"
		if lowercaseCount > 0 {
			expectedCase = "lowercase"
		}

		for _, child := range ast.Children {
			instruction := child.Value
			if instruction == "" {
				continue
			}

			isUppercase := instruction == strings.ToUpper(instruction)
			isLowercase := instruction == strings.ToLower(instruction)

			// If it's neither pure uppercase nor pure lowercase, it's mixed case
			if !isUppercase && !isLowercase {
				rules = append(rules, Rule{
					StartLine:   child.StartLine,
					EndLine:     child.EndLine,
					Code:        "ConsistentInstructionCasing",
					Description: "Instruction '" + instruction + "' should be consistently cased as " + expectedCase,
					Url:         "https://docs.docker.com/reference/build-checks/consistent-instruction-casing/",
					Severity:    SeverityWarning,
				})
			}
		}

		return rules
	}

	// There's a mix of uppercase and lowercase - flag the minority
	preferUppercase := uppercaseCount >= lowercaseCount

	for _, child := range ast.Children {
		instruction := child.Value
		if instruction == "" {
			continue
		}

		isUppercase := instruction == strings.ToUpper(instruction)
		isLowercase := instruction == strings.ToLower(instruction)

		// Flag instructions that don't match the predominant style
		if preferUppercase && !isUppercase {
			expectedStyle := "uppercase"
			rules = append(rules, Rule{
				StartLine:   child.StartLine,
				EndLine:     child.EndLine,
				Code:        "ConsistentInstructionCasing",
				Description: "Instruction '" + instruction + "' should be consistently cased as " + expectedStyle,
				Url:         "https://docs.docker.com/reference/build-checks/consistent-instruction-casing/",
				Severity:    SeverityWarning,
			})
		} else if !preferUppercase && !isLowercase {
			expectedStyle := "lowercase"
			rules = append(rules, Rule{
				StartLine:   child.StartLine,
				EndLine:     child.EndLine,
				Code:        "ConsistentInstructionCasing",
				Description: "Instruction '" + instruction + "' should be consistently cased as " + expectedStyle,
				Url:         "https://docs.docker.com/reference/build-checks/consistent-instruction-casing/",
				Severity:    SeverityWarning,
			})
		}
	}

	return rules
}

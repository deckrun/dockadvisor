package parse

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

const invalidInstructionCode = "InvalidInstruction"

// Severity represents the severity level of a rule violation
type Severity string

const (
	SeverityFatal   Severity = "fatal"
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Result struct {
	Rules []Rule `json:"rules"`
	Score int    `json:"score"`
}

type Rule struct {
	StartLine   int      // the line in the original dockerfile where the rule starts
	EndLine     int      // the line in the original dockerfile where the rule ends
	Code        string   `json:"code"`
	Description string   `json:"description"`
	Url         string   `json:"url"`
	Severity    Severity `json:"severity"`
}

// NewErrorRule creates a new Rule with error severity
func NewErrorRule(node *parser.Node, code, description, url string) Rule {
	return Rule{
		StartLine:   node.StartLine,
		EndLine:     node.EndLine,
		Code:        code,
		Description: description,
		Url:         url,
		Severity:    SeverityError,
	}
}

// NewWarningRule creates a new Rule with warning severity
func NewWarningRule(node *parser.Node, code, description, url string) Rule {
	return Rule{
		StartLine:   node.StartLine,
		EndLine:     node.EndLine,
		Code:        code,
		Description: description,
		Url:         url,
		Severity:    SeverityWarning,
	}
}

// NewFatalRule creates a new Rule with fatal severity
func NewFatalRule(node *parser.Node, code, description, url string) Rule {
	return Rule{
		StartLine:   node.StartLine,
		EndLine:     node.EndLine,
		Code:        code,
		Description: description,
		Url:         url,
		Severity:    SeverityFatal,
	}
}

func ParseDockerfile(dockerfileContent string) (*Result, error) {
	dockerfile := bytes.NewBufferString(dockerfileContent)
	result, err := parser.Parse(dockerfile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dockerfile: %v", err)
	}

	var parseRules []Rule

	// Convert parser warnings to rules
	for _, warning := range result.Warnings {
		// Skip empty continuation warnings since we have a dedicated check for those
		if strings.Contains(warning.URL, "no-empty-continuation") {
			continue
		}

		startLine := 0
		endLine := 0
		if warning.Location != nil {
			startLine = warning.Location.Start.Line
			endLine = warning.Location.End.Line
		}

		parseRules = append(parseRules, Rule{
			StartLine:   startLine,
			EndLine:     endLine,
			Code:        "ParserWarning",
			Description: warning.Short,
			Url:         warning.URL,
			Severity:    SeverityWarning,
		})
	}

	// Check for empty continuation lines (applies to all instructions)
	continuationRules := checkEmptyContinuations(dockerfileContent)
	if len(continuationRules) != 0 {
		parseRules = append(parseRules, continuationRules...)
	}

	// Check for consistent instruction casing (applies to all instructions)
	casingRules := checkConsistentInstructionCasing(result.AST)
	if len(casingRules) != 0 {
		parseRules = append(parseRules, casingRules...)
	}

	// Check for duplicate stage names (applies to all FROM instructions)
	duplicateStageRules := checkDuplicateStageNames(result.AST)
	if len(duplicateStageRules) != 0 {
		parseRules = append(parseRules, duplicateStageRules...)
	}

	// Check for constant platform flags in FROM instructions (global check)
	platformConstRules := checkPlatformFlagConstDisallowed(result.AST)
	if len(platformConstRules) != 0 {
		parseRules = append(parseRules, platformConstRules...)
	}

	// Check for JSON args recommendation in CMD/ENTRYPOINT (global check)
	jsonArgsRules := checkJSONArgsRecommended(result.AST)
	if len(jsonArgsRules) != 0 {
		parseRules = append(parseRules, jsonArgsRules...)
	}

	// Check for undefined ARG references in FROM instructions
	undefinedArgRules := checkUndefinedArgInFrom(result.AST)
	if len(undefinedArgRules) != 0 {
		parseRules = append(parseRules, undefinedArgRules...)
	}

	// Check for undefined variables across all instructions
	undefinedVarRules := checkUndefinedVar(result.AST)
	if len(undefinedVarRules) != 0 {
		parseRules = append(parseRules, undefinedVarRules...)
	}

	// Check for multiple disallowed instructions (CMD, HEALTHCHECK, ENTRYPOINT)
	multipleInstructionsRules := checkMultipleInstructionsDisallowed(result.AST)
	if len(multipleInstructionsRules) != 0 {
		parseRules = append(parseRules, multipleInstructionsRules...)
	}

	// Check for secrets in ARG or ENV instructions
	secretsRules := checkSecretsInArgOrEnv(result.AST)
	if len(secretsRules) != 0 {
		parseRules = append(parseRules, secretsRules...)
	}

	// Check for invalid default ARG values in FROM instructions
	invalidDefaultArgRules := checkInvalidDefaultArgInFrom(result.AST)
	if len(invalidDefaultArgRules) != 0 {
		parseRules = append(parseRules, invalidDefaultArgRules...)
	}

	for _, child := range result.AST.Children {
		instruction := child.Value
		insUppercase := strings.ToUpper(instruction)
		var insRules []Rule
		switch {
		case insUppercase == "FROM":
			insRules = parseFROM(child)
		case insUppercase == "WORKDIR":
			insRules = parseWorkdir(child)
		case insUppercase == "RUN":
			insRules = parseRun(child)
		case insUppercase == "EXPOSE":
			insRules = parseEXPOSE(child)
		case insUppercase == "CMD":
			insRules = parseCMD(child)
		case insUppercase == "ENTRYPOINT":
			insRules = parseENTRYPOINT(child)
		case insUppercase == "SHELL":
			insRules = parseSHELL(child)
		case insUppercase == "VOLUME":
			insRules = parseVOLUME(child)
		case insUppercase == "USER":
			insRules = parseUSER(child)
		case insUppercase == "LABEL":
			insRules = parseLABEL(child)
		case insUppercase == "ENV":
			insRules = parseENV(child)
		case insUppercase == "ARG":
			insRules = parseARG(child)
		case insUppercase == "COPY":
			insRules = parseCOPY(child)
		case insUppercase == "ADD":
			insRules = parseADD(child)
		case insUppercase == "HEALTHCHECK":
			insRules = parseHEALTHCHECK(child)
		case insUppercase == "ONBUILD":
			insRules = parseONBUILD(child)
		case insUppercase == "STOPSIGNAL":
			insRules = parseSTOPSIGNAL(child)
		case insUppercase == "MAINTAINER":
			insRules = parseMAINTAINER(child)
		default:
			// Unrecognized instruction
			insRules = []Rule{NewFatalRule(child, "UnrecognizedInstruction",
				fmt.Sprintf("'%s' is not a recognized Dockerfile instruction", instruction),
				"https://docs.docker.com/reference/dockerfile/")}
		}

		if len(insRules) != 0 {
			parseRules = append(parseRules, insRules...)
		}
	}

	score := calculateScore(parseRules)
	return &Result{Rules: parseRules, Score: score}, nil
}

func invalidInstructionRule(node *parser.Node, description string) Rule {
	return NewErrorRule(node, invalidInstructionCode, description, "")
}

// calculateScore calculates the Dockerfile score based on rule violations
// Score = 100 - (errors × 15 + warnings × 5), minimum 0
// If any fatal rule is found, score is 0
func calculateScore(rules []Rule) int {
	var errorCount, warningCount int
	for _, rule := range rules {
		if rule.Severity == SeverityFatal {
			return 0
		} else if rule.Severity == SeverityError {
			errorCount++
		} else if rule.Severity == SeverityWarning {
			warningCount++
		}
	}

	score := 100 - (errorCount*15 + warningCount*5)
	if score < 0 {
		return 0
	}
	return score
}

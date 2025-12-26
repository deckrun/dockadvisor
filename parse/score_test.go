package parse

import (
	"testing"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/stretchr/testify/require"
)

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name          string
		rules         []Rule
		expectedScore int
	}{
		{
			name:          "perfect dockerfile with no rules",
			rules:         []Rule{},
			expectedScore: 100,
		},
		{
			name: "single error rule",
			rules: []Rule{
				{Code: "InvalidInstruction", Severity: SeverityError},
			},
			expectedScore: 85, // 100 - 15 = 85
		},
		{
			name: "single warning rule",
			rules: []Rule{
				{Code: "WorkdirRelativePath", Severity: SeverityWarning},
			},
			expectedScore: 95, // 100 - 5 = 95
		},
		{
			name: "multiple errors",
			rules: []Rule{
				{Code: "InvalidInstruction", Severity: SeverityError},
				{Code: "UndefinedVar", Severity: SeverityError},
				{Code: "DuplicateStageName", Severity: SeverityError},
			},
			expectedScore: 55, // 100 - (3 × 15) = 55
		},
		{
			name: "multiple warnings",
			rules: []Rule{
				{Code: "WorkdirRelativePath", Severity: SeverityWarning},
				{Code: "FromAsCasing", Severity: SeverityWarning},
				{Code: "StageNameCasing", Severity: SeverityWarning},
			},
			expectedScore: 85, // 100 - (3 × 5) = 85
		},
		{
			name: "mixed errors and warnings",
			rules: []Rule{
				{Code: "InvalidInstruction", Severity: SeverityError},
				{Code: "WorkdirRelativePath", Severity: SeverityWarning},
				{Code: "UndefinedVar", Severity: SeverityError},
				{Code: "FromAsCasing", Severity: SeverityWarning},
			},
			expectedScore: 60, // 100 - (2 × 15) - (2 × 5) = 60
		},
		{
			name: "score should not go below zero - many errors",
			rules: []Rule{
				{Code: "Error1", Severity: SeverityError},
				{Code: "Error2", Severity: SeverityError},
				{Code: "Error3", Severity: SeverityError},
				{Code: "Error4", Severity: SeverityError},
				{Code: "Error5", Severity: SeverityError},
				{Code: "Error6", Severity: SeverityError},
				{Code: "Error7", Severity: SeverityError},
			},
			expectedScore: 0, // 100 - (7 × 15) = -5, capped at 0
		},
		{
			name: "score should not go below zero - many warnings",
			rules: []Rule{
				{Code: "Warning1", Severity: SeverityWarning},
				{Code: "Warning2", Severity: SeverityWarning},
				{Code: "Warning3", Severity: SeverityWarning},
				{Code: "Warning4", Severity: SeverityWarning},
				{Code: "Warning5", Severity: SeverityWarning},
				{Code: "Warning6", Severity: SeverityWarning},
				{Code: "Warning7", Severity: SeverityWarning},
				{Code: "Warning8", Severity: SeverityWarning},
				{Code: "Warning9", Severity: SeverityWarning},
				{Code: "Warning10", Severity: SeverityWarning},
				{Code: "Warning11", Severity: SeverityWarning},
				{Code: "Warning12", Severity: SeverityWarning},
				{Code: "Warning13", Severity: SeverityWarning},
				{Code: "Warning14", Severity: SeverityWarning},
				{Code: "Warning15", Severity: SeverityWarning},
				{Code: "Warning16", Severity: SeverityWarning},
				{Code: "Warning17", Severity: SeverityWarning},
				{Code: "Warning18", Severity: SeverityWarning},
				{Code: "Warning19", Severity: SeverityWarning},
				{Code: "Warning20", Severity: SeverityWarning},
				{Code: "Warning21", Severity: SeverityWarning},
			},
			expectedScore: 0, // 100 - (21 × 5) = -5, capped at 0
		},
		{
			name: "score should not go below zero - mixed",
			rules: []Rule{
				{Code: "Error1", Severity: SeverityError},
				{Code: "Error2", Severity: SeverityError},
				{Code: "Error3", Severity: SeverityError},
				{Code: "Error4", Severity: SeverityError},
				{Code: "Warning1", Severity: SeverityWarning},
				{Code: "Warning2", Severity: SeverityWarning},
				{Code: "Warning3", Severity: SeverityWarning},
				{Code: "Warning4", Severity: SeverityWarning},
				{Code: "Warning5", Severity: SeverityWarning},
			},
			expectedScore: 15, // 100 - (4 × 15) - (5 × 5) = 15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateScore(tt.rules)
			require.Equal(t, tt.expectedScore, score, "Score mismatch for test case: %s", tt.name)
		})
	}
}

func TestParseDockerfileScore(t *testing.T) {
	tests := []struct {
		name          string
		dockerfile    string
		expectedScore int
	}{
		{
			name:          "perfect dockerfile",
			dockerfile:    "FROM alpine:latest\nWORKDIR /app\nCMD [\"echo\", \"hello\"]",
			expectedScore: 100,
		},
		{
			name:          "dockerfile with one warning",
			dockerfile:    "FROM alpine:latest\nWORKDIR app\n", // Relative path warning
			expectedScore: 95,                                  // 100 - 5 = 95
		},
		{
			name:          "dockerfile with one error",
			dockerfile:    "FROM\n", // Missing image (error)
			expectedScore: 85,       // 100 - 15 = 85
		},
		{
			name: "dockerfile with multiple issues",
			dockerfile: `FROM alpine:latest
WORKDIR app
RUN
`, // Relative path (warning) + Missing command (error)
			expectedScore: 80, // 100 - 15 - 5 = 80
		},
		{
			name: "dockerfile with many warnings",
			dockerfile: `from alpine:latest as Builder
WORKDIR app
Run echo "test"
Cmd echo "test"
`, // 3 ConsistentInstructionCasing + 1 StageNameCasing + 1 WorkdirRelativePath + 1 JSONArgsRecommended = 6 warnings
			expectedScore: 70, // 100 - (6 × 5) = 70
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDockerfile(tt.dockerfile)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.expectedScore, result.Score, "Score mismatch for test case: %s", tt.name)
		})
	}
}

func TestNewErrorRule(t *testing.T) {
	node := &parser.Node{
		StartLine: 1,
		EndLine:   1,
	}

	rule := NewErrorRule(node, "TestCode", "Test description", "https://test.com")

	require.Equal(t, 1, rule.StartLine)
	require.Equal(t, 1, rule.EndLine)
	require.Equal(t, "TestCode", rule.Code)
	require.Equal(t, "Test description", rule.Description)
	require.Equal(t, "https://test.com", rule.Url)
	require.Equal(t, SeverityError, rule.Severity)
}

func TestNewWarningRule(t *testing.T) {
	node := &parser.Node{
		StartLine: 2,
		EndLine:   3,
	}

	rule := NewWarningRule(node, "WarningCode", "Warning description", "https://warning.com")

	require.Equal(t, 2, rule.StartLine)
	require.Equal(t, 3, rule.EndLine)
	require.Equal(t, "WarningCode", rule.Code)
	require.Equal(t, "Warning description", rule.Description)
	require.Equal(t, "https://warning.com", rule.Url)
	require.Equal(t, SeverityWarning, rule.Severity)
}

func TestInvalidInstructionRule(t *testing.T) {
	node := &parser.Node{
		StartLine: 5,
		EndLine:   5,
	}

	rule := invalidInstructionRule(node, "Test invalid instruction")

	require.Equal(t, 5, rule.StartLine)
	require.Equal(t, 5, rule.EndLine)
	require.Equal(t, invalidInstructionCode, rule.Code)
	require.Equal(t, "Test invalid instruction", rule.Description)
	require.Equal(t, "", rule.Url)
	require.Equal(t, SeverityError, rule.Severity)
}

func TestUnrecognizedInstructionScore(t *testing.T) {
	dockerfile := `FROM alpine:latest
WORKDIR /app
FOOBAR invalid
CMD ["echo", "hello"]`

	result, err := ParseDockerfile(dockerfile)
	require.NoError(t, err)
	require.Equal(t, 0, result.Score, "Dockerfile with UnrecognizedInstruction should have score 0")
}

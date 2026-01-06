# Contributing to Dockadvisor

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## Guidelines

- Follow [Effective Go](https://go.dev/doc/effective_go) practices
- Write tests for all new functionality
- Use `gofmt` and `golint` before committing
- Keep commits focused and atomic
- Write clear commit messages

## Development

### Prerequisites

- Go 1.25.3 or later
- Make (optional, for using Makefile commands)

### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run only WASM tests
make test-wasm

# Run specific test
go test -v ./parse -run TestCheckFromAsCasing
```

### Adding New Rules

#### For Instruction-Specific Rules:

1. Create or update a validator file in `parse/` (e.g., `parse/from.go`)
2. Add the rule check in `parse/parse.go` switch statement
3. Write comprehensive tests in the corresponding `_test.go` file
4. Update this README with the new rule documentation

Example validator function:

```go
func parseYourInstruction(node *parser.Node) []Rule {
    if node.Next == nil {
        return []Rule{invalidInstructionRule(node, "YOUR_INSTRUCTION requires arguments")}
    }

    var rules []Rule

    // Add your validation logic here
    if !checkYourCondition(node.Next.Value) {
        // Use NewErrorRule for build failures/invalid syntax
        rules = append(rules, NewErrorRule(node, "YourRuleCode",
            "Clear description of the issue",
            "https://docs.docker.com/reference/dockerfile/#your-instruction"))
    }

    // Or use NewWarningRule for style/best practices
    if !checkBestPractice(node.Next.Value) {
        rules = append(rules, NewWarningRule(node, "YourStyleRule",
            "Style recommendation",
            "https://docs.docker.com/build/building/best-practices/"))
    }

    return rules
}
```

#### For Global Rules:

1. Create a new file in `parse/` (e.g., `parse/your_check.go`)
2. Implement a global validation function that takes the AST or content
3. Call it from `ParseDockerfile()` in `parse/parse.go`
4. Write comprehensive tests
5. Update this README

Example global validator:

```go
func checkYourGlobalRule(ast *parser.Node) []Rule {
    var rules []Rule

    // Iterate through all instructions
    for _, child := range ast.Children {
        if shouldFlagNode(child) {
            rules = append(rules, NewWarningRule(child, "YourGlobalRule",
                "Description of the issue",
                "https://docs.docker.com/..."))
        }
    }

    return rules
}
```

## Testing

The project uses `testify/require` for clean, readable test assertions. All validation functions have comprehensive test coverage including:

- ✅ Valid cases (should pass)
- ❌ Invalid cases (should fail)
- 🔍 Edge cases

Example test structure:

```go
func TestCheckYourFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {
            name:     "valid case",
            input:    "some valid input",
            expected: true,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := checkYourFunction(tt.input)
            require.Equal(t, tt.expected, result)
        })
    }
}
```

## Dependencies

- [moby/buildkit](https://github.com/moby/buildkit) - Dockerfile parser from Docker's BuildKit
- [stretchr/testify](https://github.com/stretchr/testify) - Testing toolkit

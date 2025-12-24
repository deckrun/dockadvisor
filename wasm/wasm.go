//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/deckrun/dockadvisor/parse"
)

// parseDockerfileLogic contains the core business logic without JS dependencies
func parseDockerfileLogic(dockerfileContent string) (response map[string]any) {
	// Recover from panics (e.g., from log.Fatal calls in the parser)
	defer func() {
		if r := recover(); r != nil {
			response = map[string]any{
				"success": false,
				"error":   "parser error: " + fmt.Sprint(r),
			}
		}
	}()

	result, err := parse.ParseDockerfile(dockerfileContent)
	if err != nil {
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}
	}

	// Handle nil result (should not happen, but be defensive)
	if result == nil {
		return map[string]any{
			"success": false,
			"error":   "parser returned nil result",
		}
	}

	// Convert result to a format suitable for JavaScript
	rules := make([]any, 0, len(result.Rules))
	for _, r := range result.Rules {
		rules = append(rules, map[string]any{
			"startLine":   r.StartLine,
			"endLine":     r.EndLine,
			"code":        r.Code,
			"description": r.Description,
			"url":         r.Url,
			"severity":    string(r.Severity),
		})
	}

	return map[string]any{
		"success": true,
		"rules":   rules,
		"score":   result.Score,
	}
}

func parseDockerfile(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]any{
			"success": false,
			"error":   "missing required argument: dockerfileContent",
		}
	}
	dockerfileContent := args[0].String()
	return parseDockerfileLogic(dockerfileContent)
}

func main() {
	// Register the parseDockerfile function to be callable from JavaScript
	js.Global().Set("parseDockerfile", js.FuncOf(parseDockerfile))

	// Prevent the Go program from exiting
	select {}
}

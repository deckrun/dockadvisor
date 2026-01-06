.PHONY: build build-wasm tinybuild clean test test-wasm

# Output files
WASM_OUTPUT := dockadvisor.wasm
CLI_OUTPUT := dockadvisor

# Build the CLI binary
build:
	go build -o $(CLI_OUTPUT) ./cmd/dockadvisor

# Build the WebAssembly binary
build-wasm:
	GOOS=js GOARCH=wasm go build -o $(WASM_OUTPUT) wasm/wasm.go

# Run all tests
test: test-wasm
	go test -v ./...

# Run WASM-specific tests
test-wasm:
	GOOS=js GOARCH=wasm go test -v -exec="$$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./wasm

# Clean build artifacts
clean:
	rm -f $(WASM_OUTPUT) $(CLI_OUTPUT)


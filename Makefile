.PHONY: build tinybuild clean test test-wasm

# Output file
OUTPUT := dockadvisor.wasm

# Build the WebAssembly binary
build:
	GOOS=js GOARCH=wasm go build -o $(OUTPUT) wasm/wasm.go

# Run all tests
test: test-wasm
	go test -v ./...

# Run WASM-specific tests
test-wasm:
	GOOS=js GOARCH=wasm go test -v -exec="$$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./wasm

# Clean build artifacts
clean:
	rm -f $(OUTPUT)


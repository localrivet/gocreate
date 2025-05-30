.PHONY: all build test lint clean bench coverage install build-all build-linux build-darwin build-windows clean-dist release

# Default target
all: build test lint

# Build the MCP server
build:
	go build -v -o gocreate .

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	go clean
	rm -f gocreate
	rm -rf dist/

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run all checks
check: fmt vet lint test

# Build for multiple platforms
build-all: clean-dist
	@echo "Building cross-platform binaries..."
	@mkdir -p dist/linux-amd64 dist/linux-arm64 dist/darwin-amd64 dist/darwin-arm64 dist/windows-amd64 dist/windows-arm64
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/gocreate .
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/gocreate .
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/gocreate .
	GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/gocreate .
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/gocreate.exe .
	GOOS=windows GOARCH=arm64 go build -o dist/windows-arm64/gocreate.exe .
	@echo "Cross-platform binaries built in dist/ directory"

build-linux:
	@mkdir -p dist/linux-amd64 dist/linux-arm64
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/gocreate .
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/gocreate .

build-darwin:
	@mkdir -p dist/darwin-amd64 dist/darwin-arm64
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/gocreate .
	GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/gocreate .

build-windows:
	@mkdir -p dist/windows-amd64 dist/windows-arm64
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/gocreate.exe .
	GOOS=windows GOARCH=arm64 go build -o dist/windows-arm64/gocreate.exe .

# Clean dist directory
clean-dist:
	rm -rf dist/

# Release build with version and archives
release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Usage: make release VERSION=v1.0.0"; exit 1; fi
	@echo "Building release $(VERSION)..."
	@mkdir -p dist/linux-amd64 dist/linux-arm64 dist/darwin-amd64 dist/darwin-arm64 dist/windows-amd64 dist/windows-arm64
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/linux-amd64/gocreate .
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/linux-arm64/gocreate .
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/darwin-amd64/gocreate .
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/darwin-arm64/gocreate .
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/windows-amd64/gocreate.exe .
	GOOS=windows GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/windows-arm64/gocreate.exe .
	@echo "Creating archives..."
	@cd dist/linux-amd64 && tar -czf ../gocreate-$(VERSION)-linux-amd64.tar.gz gocreate
	@cd dist/linux-arm64 && tar -czf ../gocreate-$(VERSION)-linux-arm64.tar.gz gocreate
	@cd dist/darwin-amd64 && tar -czf ../gocreate-$(VERSION)-darwin-amd64.tar.gz gocreate
	@cd dist/darwin-arm64 && tar -czf ../gocreate-$(VERSION)-darwin-arm64.tar.gz gocreate
	@cd dist/windows-amd64 && zip ../gocreate-$(VERSION)-windows-amd64.zip gocreate.exe
	@cd dist/windows-arm64 && zip ../gocreate-$(VERSION)-windows-arm64.zip gocreate.exe
	@echo "Release $(VERSION) built successfully!"

# Development helpers
dev: build
	./gocreate

# Run with verbose logging
dev-verbose: build
	./gocreate --verbose 
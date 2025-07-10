.PHONY: build clean test run help install

# Variables
BINARY_NAME=universal-checker
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_MAC=$(BINARY_NAME)_mac

# Default target
all: build

# Build for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) cmd/main.go
	@echo "✅ Build complete: $(BINARY_NAME)"

# Build GUI for current platform
build-gui:
	@echo "Building $(BINARY_NAME)-gui..."
	go build -o $(BINARY_NAME)-gui cmd/gui/main.go
	@echo "✅ GUI build complete: $(BINARY_NAME)-gui"

# Build for all platforms
build-all: build-linux build-windows build-mac

# Build GUI for all platforms
build-gui-all: build-gui-linux build-gui-windows build-gui-mac

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_UNIX) cmd/main.go
	@echo "✅ Linux build complete: $(BINARY_UNIX)"

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WINDOWS) cmd/main.go
	@echo "✅ Windows build complete: $(BINARY_WINDOWS)"

# Build for macOS
build-mac:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_MAC) cmd/main.go
	@echo "✅ macOS build complete: $(BINARY_MAC)"

# Build GUI for Linux
build-gui-linux:
	@echo "Building GUI for Linux..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_UNIX)-gui cmd/gui/main.go
	@echo "✅ Linux GUI build complete: $(BINARY_UNIX)-gui"

# Build GUI for Windows
build-gui-windows:
	@echo "Building GUI for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WINDOWS:%.exe=%-gui.exe) cmd/gui/main.go
	@echo "✅ Windows GUI build complete: $(BINARY_WINDOWS:%.exe=%-gui.exe)"

# Build GUI for macOS
build-gui-mac:
	@echo "Building GUI for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_MAC)-gui cmd/gui/main.go
	@echo "✅ macOS GUI build complete: $(BINARY_MAC)-gui"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	go clean
	rm -f $(BINARY_NAME) $(BINARY_UNIX) $(BINARY_WINDOWS) $(BINARY_MAC)
	rm -rf results/
	@echo "✅ Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies updated"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...
	@echo "✅ Tests complete"

# Run with sample data
run-sample:
	@echo "Running with sample data..."
	./$(BINARY_NAME) \
		--configs configs/sample.opk \
		--combos data/combos/sample_combos.txt \
		--workers 50 \
		--auto-scrape=false \
		--request-timeout 10000

# Install dependencies
install:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "✅ Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run
	@echo "✅ Linting complete"

# Create release builds
release: clean build-all
	@echo "Creating release packages..."
	mkdir -p dist
	tar -czf dist/$(BINARY_NAME)_linux_amd64.tar.gz $(BINARY_UNIX) README.md configs/ data/
	zip -r dist/$(BINARY_NAME)_windows_amd64.zip $(BINARY_WINDOWS) README.md configs/ data/
	tar -czf dist/$(BINARY_NAME)_darwin_amd64.tar.gz $(BINARY_MAC) README.md configs/ data/
	@echo "✅ Release packages created in dist/"

# Show help
help:
	@echo "Universal Checker - Build Commands"
	@echo "=================================="
	@echo "build         - Build for current platform"
	@echo "build-all     - Build for all platforms (Linux, Windows, macOS)"
	@echo "build-linux   - Build for Linux"
	@echo "build-windows - Build for Windows"
	@echo "build-mac     - Build for macOS"
	@echo "clean         - Clean build artifacts"
	@echo "deps          - Download and update dependencies"
	@echo "test          - Run tests"
	@echo "run-sample    - Run with sample configuration"
	@echo "install       - Install dependencies"
	@echo "fmt           - Format code"
	@echo "lint          - Lint code"
	@echo "release       - Create release packages"
	@echo "help          - Show this help message"

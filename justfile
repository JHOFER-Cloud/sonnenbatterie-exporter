# Default recipe to display help
default:
    @just --list

# Build the binary
build:
    @echo "Building sonnenbatterie-exporter..."
    go build -o sonnenbatterie-exporter .
    @echo "✓ Build complete: ./sonnenbatterie-exporter"

# Run the exporter locally (requires SONNENBATTERIE_API_URL)
run:
    @echo "Starting sonnenbatterie-exporter..."
    go run .

# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

# Run tests with race detection
test-race:
    @echo "Running tests with race detection..."
    go test -v -race ./...

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    go test -v -coverprofile=coverage.out -covermode=atomic ./...
    go tool cover -func=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    @echo "✓ Coverage report: coverage.html"
    @echo "✓ Open coverage.html in your browser to view detailed coverage"

# Format code
fmt:
    @echo "Formatting code..."
    gofumpt -w .
    goimports-reviser .
    @echo "✓ Code formatted"

# Run linter
lint:
    @echo "Running linter..."
    golangci-lint run ./...

# Tidy dependencies
tidy:
    @echo "Tidying dependencies..."
    go mod tidy
    @echo "✓ Dependencies tidied"

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    go mod download
    @echo "✓ Dependencies downloaded"

# Build Docker image
docker-build tag="latest":
    @echo "Building Docker image..."
    docker build -t sonnenbatterie-exporter:{{tag}} .
    @echo "✓ Docker image built: sonnenbatterie-exporter:{{tag}}"

# Run Docker container (requires sonnen-batterie-api running)
docker-run port="9090" api_url="http://host.docker.internal:8080":
    @echo "Running Docker container..."
    @echo "NOTE: Ensure sonnen-batterie-api is running at {{api_url}}"
    docker run --rm -p {{port}}:9090 \
        -e SONNENBATTERIE_API_URL={{api_url}} \
        sonnenbatterie-exporter:latest

# Run both API and exporter with docker compose (for local testing)
docker-compose-up:
    @echo "ERROR: docker-compose not provided"
    @echo "Please run sonnenbatterie-api separately:"
    @echo "  docker run -d -p 8080:8080 \\"
    @echo "    -e SONNENBATTERIE_IP='192.168.1.100' \\"
    @echo "    -e SONNENBATTERIE_USER_NAME='User' \\"
    @echo "    -e SONNENBATTERIE_USER_PASSWORD='your-password' \\"
    @echo "    larmic/sonnen-batterie-api:latest"
    @echo ""
    @echo "Then run: just docker-run"

# Stop all running containers
docker-stop:
    @echo "Stopping all sonnenbatterie-exporter containers..."
    docker ps -q --filter ancestor=sonnenbatterie-exporter | xargs -r docker stop

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -f sonnenbatterie-exporter
    rm -f coverage.out coverage.html
    @echo "✓ Clean complete"

# Clean all (including Docker images)
clean-all: clean
    @echo "Removing Docker images..."
    docker images sonnenbatterie-exporter -q | xargs -r docker rmi -f
    @echo "✓ All clean"

# Install required tools
install-tools:
    @echo "Installing development tools..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "✓ Tools installed"

# Run development server with hot reload (requires air)
dev:
    @echo "Starting development server with hot reload..."
    @if ! command -v air > /dev/null 2>&1; then \
        echo "Installing air..."; \
        go install github.com/air-verse/air@latest; \
    fi
    @$(go env GOPATH)/bin/air

# Verify build for multiple platforms
verify-build:
    @echo "Verifying build for multiple platforms..."
    GOOS=linux GOARCH=amd64 go build -o /dev/null .
    GOOS=linux GOARCH=arm64 go build -o /dev/null .
    GOOS=darwin GOARCH=amd64 go build -o /dev/null .
    GOOS=darwin GOARCH=arm64 go build -o /dev/null .
    @echo "✓ All platform builds verified"

# Show project info
info:
    @echo "=== SonnenBatterie Exporter ==="
    @echo "Go version: $(go version)"
    @echo "Project: $(go list -m)"
    @echo "Dependencies:"
    @go list -m all | grep -v "$(go list -m)" | head -10

# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed to be able to call HTTPS endpoints)
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create appuser for security
RUN adduser -D -g '' appuser

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o luma ./cmd/main.go

# Runtime stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy our static executable
COPY --from=builder /build/luma /luma

# Use an unprivileged user
USER appuser

# Create volume for data
VOLUME ["/data"]

# Create volume for logs
VOLUME ["/logs"]

# Expose port for potential web interface
EXPOSE 8080

# Set working directory
WORKDIR /data

# Command to run
ENTRYPOINT ["/luma"]

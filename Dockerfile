# Build stage
FROM golang:1.21-alpine AS builder

# Install ca-certificates for SSL
RUN apk add --no-cache ca-certificates git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o ck-cli \
    ./cmd/ck-cli

# Final stage
FROM scratch

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/ck-cli /ck-cli

# Set the entrypoint
ENTRYPOINT ["/ck-cli"]
# Build stage
FROM golang:1.24-alpine AS builder

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docs-gen .

# Final stage - scratch container
FROM scratch

# Copy ca-certificates from builder stage for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/docs-gen /docs-gen

# Set the entrypoint
ENTRYPOINT ["/docs-gen"]
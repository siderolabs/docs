# Build stage
FROM golang:1.26-alpine AS builder

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies (none currently, but keeping for future)
RUN go mod download || true

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o omni-config-gen .

# Final stage - scratch container
FROM scratch

# Copy ca-certificates from builder stage for HTTPS requests (schema is fetched over HTTPS)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/omni-config-gen /omni-config-gen

# Set the entrypoint
ENTRYPOINT ["/omni-config-gen"]

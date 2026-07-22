# Build stage
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy go mod files (no external dependencies currently).
COPY go.mod ./
RUN go mod download || true

# Copy source code and the embedded exceptions list.
COPY main.go exceptions.txt ./

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o style-guide-checker .

# Final stage - scratch container
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/style-guide-checker /style-guide-checker

ENTRYPOINT ["/style-guide-checker"]

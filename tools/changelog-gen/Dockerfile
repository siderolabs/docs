# Build stage
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o changelog-gen .

# Final stage - scratch container
FROM scratch

# ca-certificates are needed for HTTPS requests to the GitHub API.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/changelog-gen /changelog-gen

ENTRYPOINT ["/changelog-gen"]

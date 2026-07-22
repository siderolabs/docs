# Downloader stage: fetch the pinned omnictl release binary.
FROM alpine:3.21 AS downloader

# omnictl is released from the siderolabs/omni repository.
ARG OMNICTL_VERSION=v1.9.0
# Populated automatically by BuildKit/buildx (e.g. amd64, arm64).
ARG TARGETARCH=amd64

RUN apk --no-cache add ca-certificates curl \
    && curl -fsSL -o /omnictl \
        "https://github.com/siderolabs/omni/releases/download/${OMNICTL_VERSION}/omnictl-linux-${TARGETARCH}" \
    && chmod +x /omnictl

# Final stage. Alpine (not scratch) so callers can run omnictl and read its
# generated files inside the container, then stream them to stdout. This avoids
# writing generated docs into a bind-mounted host directory, which is unreliable
# on Docker Desktop for macOS.
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

COPY --from=downloader /omnictl /usr/local/bin/omnictl

ENTRYPOINT ["omnictl"]

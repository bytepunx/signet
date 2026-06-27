# syntax=docker/dockerfile:1
# Multi-stage build for signet.
# Targets:
#   signetd  — the server daemon (default)
#   signet   — the operator CLI

FROM golang:1.26-alpine AS builder

# Install CA certificates for HTTPS calls during go mod download.
RUN apk add --no-cache ca-certificates git

WORKDIR /src

# Cache module downloads separately from source so a code change doesn't
# invalidate the dependency layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build signetd — fully static, no CGO.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/signetd \
    ./cmd/signetd

# Build signet CLI — same flags.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/signet \
    ./cmd/signet

# ---------------------------------------------------------------------------
# signetd runtime image
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS signetd

COPY --from=builder /out/signetd /signetd

# SPIRE agent socket lives at a host path mounted into the pod; no filesystem
# writes are required at runtime.
EXPOSE 8443 8445

USER nonroot:nonroot

ENTRYPOINT ["/signetd"]

# ---------------------------------------------------------------------------
# signet CLI image (useful for operator tooling in-cluster)
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS signet

COPY --from=builder /out/signet /signet

USER nonroot:nonroot

ENTRYPOINT ["/signet"]

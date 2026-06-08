# ── Stage 1: Build ────────────────────────────────────────────────────────────
# Use the full Go image so go.work / workspace builds work correctly.
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy workspace descriptor.
COPY go.work ./

# Copy ALL go.mod files (go work sync needs them all to resolve the workspace).
COPY shared/go.mod shared/go.mod
COPY api/go.mod api/go.mod
COPY workers/crawler/go.mod workers/crawler/go.mod
COPY workers/rank-tracker/go.mod workers/rank-tracker/go.mod

# Download dependencies for the api module (uses workspace for replace directives).
RUN cd api && go mod download

# Copy only the relevant source.
COPY shared/ shared/
COPY api/ api/

# Build a fully static binary — no libc required.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build \
    -ldflags="-s -w -extldflags=-static" \
    -trimpath \
    -o /seoxpert-api \
    ./api/cmd/server

# ── Stage 2: Run ──────────────────────────────────────────────────────────────
# distroless/static is ~2MB — no shell, no package manager, minimal attack surface.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /seoxpert-api /seoxpert-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/seoxpert-api"]

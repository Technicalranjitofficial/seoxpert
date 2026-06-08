# ── Stage 1: Builder ─────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy go.work and all module go.mod files first (layer cache)
COPY go.work go.work.sum* ./
COPY shared/go.mod shared/go.sum* ./shared/
COPY api/go.mod api/go.sum* ./api/
COPY workers/crawler/go.mod workers/crawler/go.sum* ./workers/crawler/
COPY workers/rank-tracker/go.mod workers/rank-tracker/go.sum* ./workers/rank-tracker/

# Download dependencies for rank-tracker only
RUN cd workers/rank-tracker && go mod download

# Copy source
COPY shared/ ./shared/
COPY workers/rank-tracker/ ./workers/rank-tracker/

# Build static binary for ARM64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags="-s -w" -o /rank-tracker \
    ./workers/rank-tracker/cmd/main.go

# ── Stage 2: Runtime (chromedp headless-shell) ────────────────────────────────
FROM chromedp/headless-shell:latest AS runner

COPY --from=builder /rank-tracker /rank-tracker

ENTRYPOINT ["/rank-tracker"]

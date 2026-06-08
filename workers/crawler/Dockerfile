# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.work go.work.sum* ./
COPY shared/go.mod shared/go.mod
COPY workers/crawler/go.mod workers/crawler/go.mod

RUN go work sync

COPY shared/ shared/
COPY workers/crawler/ workers/crawler/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /crawler-worker \
    ./workers/crawler/cmd

# ── Stage 2: Run ──────────────────────────────────────────────────────────────
# chromedp needs a real Chrome/Chromium binary — use a slim Chromium image.
FROM chromedp/headless-shell:latest

COPY --from=builder /crawler-worker /crawler-worker
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/crawler-worker"]

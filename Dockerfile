# syntax=docker/dockerfile:1

# ---- Build stage -----------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Download dependencies first so they are cached across source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/phoenix-logs-rmq \
    ./cmd/phoenix-logs-rmq

# ---- Runtime stage ---------------------------------------------------------
FROM alpine:3.22

ARG VERSION=dev
LABEL org.opencontainers.image.title="Phoenix-Logs-RMQ" \
      org.opencontainers.image.description="Consumes audit and mail-flow logs from RabbitMQ and bulk-inserts them into ClickHouse." \
      org.opencontainers.image.source="https://github.com/Yukthi-Systems/Phoenix-Logs-RMQ" \
      org.opencontainers.image.url="https://hub.docker.com/r/rjyspl/phoenix-logs-rmq" \
      org.opencontainers.image.vendor="Yukthi Systems Private Limited" \
      org.opencontainers.image.licenses="GPL-3.0-only" \
      org.opencontainers.image.version="${VERSION}"

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app/logs

WORKDIR /app
COPY --from=builder /out/phoenix-logs-rmq /usr/local/bin/phoenix-logs-rmq

# Logs go to the rotating file in /app/logs. Set LOG_CONSOLE=true to also
# write them to stderr for `docker logs`.
ENV LOG_FILE=/app/logs/app.log \
    LOG_CONSOLE=false \
    LOG_LEVEL=info

VOLUME ["/app/logs"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD pgrep -x phoenix-logs-rmq > /dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/phoenix-logs-rmq"]

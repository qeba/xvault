# syntax=docker/dockerfile:1

FROM golang:1.25.5-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM debian:bookworm-slim
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates; \
    rm -rf /var/lib/apt/lists/*; \
    # Create worker user with proper permissions
    useradd -m -u 1000 -s /sbin/nologin worker; \
    # Create storage directory with proper permissions
    mkdir -p /var/lib/xvault/backups; \
    chown -R worker:worker /var/lib/xvault/backups; \
    chmod 755 /var/lib/xvault/backups
WORKDIR /
COPY --from=build /out/worker /worker
USER worker
ENTRYPOINT ["/worker"]

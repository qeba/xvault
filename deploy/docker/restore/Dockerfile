# syntax=docker/dockerfile:1

FROM golang:1.25.5-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/restore ./cmd/restore

FROM debian:bookworm-slim
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates; \
    rm -rf /var/lib/apt/lists/*; \
    # Create restore user with proper permissions
    useradd -m -u 1001 -s /sbin/nologin restore; \
    # Create downloads directory with proper permissions
    mkdir -p /var/lib/xvault/downloads; \
    chown -R restore:restore /var/lib/xvault/downloads; \
    chmod 755 /var/lib/xvault/downloads
WORKDIR /
COPY --from=build /out/restore /restore
USER restore
ENTRYPOINT ["/restore"]

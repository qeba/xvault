# syntax=docker/dockerfile:1

FROM golang:1.25.5-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/hub ./cmd/hub

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/hub /hub
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/hub"]

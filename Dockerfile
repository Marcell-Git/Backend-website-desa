# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata postgresql17-client \
    && addgroup -S app && adduser -S app -G app

ENV TZ=Asia/Jakarta
WORKDIR /app

COPY --from=builder --chown=app:app /out/server ./server
RUN mkdir -p /app/uploads /app/backups && chown -R app:app /app

USER app
EXPOSE 8080
ENTRYPOINT ["./server"]

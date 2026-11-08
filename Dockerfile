# Multi-stage Dockerfile for AdytumFama (Collector, Exporter, and Web Server)
# syntax=docker/dockerfile:1

ARG NODE_VERSION=22
ARG GO_VERSION=1.24

# 1. Frontend Web Builder
FROM node:${NODE_VERSION}-alpine AS web-builder

WORKDIR /web
RUN corepack enable && corepack prepare pnpm@latest --activate

COPY web/package.json web/pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile || pnpm install

COPY web/ ./
RUN pnpm build

# 2. Go Binaries Builder
FROM golang:${GO_VERSION}-alpine AS go-builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy exported frontend assets before compiling server (for embed.FS)
COPY --from=web-builder /web/out ./web/out

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/collector ./cmd/collector
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/exporter ./cmd/exporter
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/server ./cmd/server

# 3. Final lightweight runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl bash

# Install AWS IAM Roles Anywhere signing helper binary
ARG AWS_SIGNING_HELPER_VERSION=1.4.0
RUN curl -sSfL "https://rolesanywhere.amazonaws.com/releases/${AWS_SIGNING_HELPER_VERSION}/X86_64/Linux/aws_signing_helper" \
    -o /usr/local/bin/aws_signing_helper && \
    chmod +x /usr/local/bin/aws_signing_helper

WORKDIR /app

COPY --from=go-builder /bin/collector /usr/local/bin/collector
COPY --from=go-builder /bin/exporter /usr/local/bin/exporter
COPY --from=go-builder /bin/server /usr/local/bin/server

# Default non-root user
USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["server"]

# Multi-stage Dockerfile for AdytumFama Collector and Exporter
# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build collector and exporter binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/collector ./cmd/collector
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/exporter ./cmd/exporter

# Final lightweight runtime image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl bash

# Install AWS IAM Roles Anywhere signing helper binary
ARG AWS_SIGNING_HELPER_VERSION=1.4.0
RUN curl -sSfL "https://rolesanywhere.amazonaws.com/releases/${AWS_SIGNING_HELPER_VERSION}/X86_64/Linux/aws_signing_helper" \
    -o /usr/local/bin/aws_signing_helper && \
    chmod +x /usr/local/bin/aws_signing_helper

WORKDIR /app

COPY --from=builder /bin/collector /usr/local/bin/collector
COPY --from=builder /bin/exporter /usr/local/bin/exporter

# Default non-root user
USER 65534:65534

ENTRYPOINT ["collector"]

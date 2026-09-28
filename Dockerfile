# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /src

# Keep dependency downloads cached when only application sources change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG REPO=github.com/brasilcep/api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w \
      -X 'main.Version=${VERSION}' \
      -X 'main.Commit=${COMMIT}' \
      -X 'main.Repo=${REPO}' \
      -X 'main.Compiler=$(go version | awk '{print $3}')'" \
    -o /out/wserver ./main.go

FROM alpine:3.22 AS final

RUN apk add --no-cache su-exec tar && \
    addgroup -S brasilcep && \
    adduser -S -G brasilcep -h /app brasilcep

WORKDIR /app

COPY --from=builder /out/wserver ./wserver
COPY --chown=brasilcep:brasilcep --chmod=755 docker-entrypoint.sh ./docker-entrypoint.sh

ENV DB_PATH=/app/data \
    MODE=listen

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/healthcheck || exit 1

ENTRYPOINT ["/app/docker-entrypoint.sh"]

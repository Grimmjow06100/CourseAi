# syntax=docker/dockerfile:1.7
FROM golang:1.27-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,id=course-ai-go-mod,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,id=course-ai-go-mod,target=/go/pkg/mod --mount=type=cache,id=course-ai-go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/course-ai-api ./cmd/api
RUN --mount=type=cache,id=course-ai-go-mod,target=/go/pkg/mod --mount=type=cache,id=course-ai-go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOBIN=/out go install \
    -tags="no_clickhouse no_mssql no_mysql no_sqlite3 no_libsql no_vertica no_ydb" \
    github.com/pressly/goose/v3/cmd/goose@v3.27.1

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S courseai \
    && adduser -S -G courseai courseai

WORKDIR /app
COPY --from=builder /out/course-ai-api /app/course-ai-api
COPY --from=builder /out/goose /app/goose
COPY --chown=courseai:courseai migrations /app/migrations
COPY --chown=courseai:courseai prompts /app/prompts

ENV GIN_MODE=release \
    PROMPTS_DIR=/app/prompts
USER courseai
EXPOSE 8080

ENTRYPOINT ["/app/course-ai-api"]

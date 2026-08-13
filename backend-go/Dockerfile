FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/course-ai-api ./cmd/api
RUN CGO_ENABLED=0 GOBIN=/out go install github.com/pressly/goose/v3/cmd/goose@v3.27.1

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

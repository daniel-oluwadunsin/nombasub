# syntax=docker/dockerfile:1.7

# ---------- builder ----------
FROM golang:1.25-alpine AS builder

WORKDIR /src

# CA certs + tzdata for HTTPS calls and cron scheduling.
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries. Statically linked so the runtime image can be minimal.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/mcp ./cmd/mcp

# ---------- runtime ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl \
 && addgroup -S nombasub && adduser -S nombasub -G nombasub

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY --from=builder /out/mcp /app/mcp

USER nombasub

EXPOSE 8080 8081

# Default entrypoint is the API. docker-compose sets `command: /app/mcp` on the
# MCP service to run the second binary from the same image.
CMD ["/app/api"]

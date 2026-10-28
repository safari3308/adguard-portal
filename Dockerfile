# Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
ENV CI=true
RUN corepack enable && corepack prepare pnpm@10 --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# Build Backend (with CGO for high-performance SQLite)
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built assets into web/dist
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Final Runtime Container
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

COPY --from=backend-builder --chown=appuser:appgroup /app/server /app/server

USER appuser
EXPOSE 8080

ENV PORT=8080 \
    DB_PATH=/app/data/portal.db \
    DOH_DOMAIN=localhost

VOLUME ["/app/data"]
ENTRYPOINT ["/app/server"]

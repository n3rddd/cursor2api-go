# ---- Builder stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Install build deps
RUN apk add --no-cache git ca-certificates

# Cache go modules layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/cursor2api-go .

# ---- Runtime stage ----
FROM alpine:3.21

RUN apk --no-cache add ca-certificates nodejs

WORKDIR /app

# Copy binary and required assets
COPY --from=builder /out/cursor2api-go .
COPY static ./static
COPY jscode ./jscode

# Create non-root user
RUN adduser -D -g '' appuser && \
    chown -R appuser:appuser /app
USER appuser

EXPOSE 8002

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8002/health || exit 1

CMD ["./cursor2api-go"]

# Build stage
FROM golang:1.24-bookworm AS builder

# Install gcc for CGO (required by mattn/go-sqlite3)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o cardman ./cmd/main.go

# Runtime stage
FROM debian:bookworm-slim

# Install SQLite runtime library and CA certs
RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-0 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 1001 cardman

WORKDIR /app

# Copy binary
COPY --from=builder /build/cardman /usr/local/bin/cardman

# Data directory for SQLite database and config
RUN mkdir -p /app/data && chown cardman:cardman /app/data

USER cardman

VOLUME ["/app/data"]

ENTRYPOINT ["cardman"]

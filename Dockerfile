# Build stage
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build (pure Go: modernc.org/sqlite needs no CGO)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o cardman ./cmd/main.go

# Runtime stage
FROM debian:bookworm-slim

# CA certs for HTTPS calls to the card APIs
RUN apt-get update && apt-get install -y --no-install-recommends \
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

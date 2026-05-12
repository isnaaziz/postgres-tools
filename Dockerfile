# Builder Stage
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o pg_migrate_tool main.go

# Final Stage
FROM debian:bookworm-slim

# Install PostgreSQL client tools
RUN apt-get update && apt-get install -y \
    postgresql-client \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/pg_migrate_tool .

# Create a directory for backups
RUN mkdir -p /app/backups

# Expose the port
EXPOSE 8765

# Run the application
CMD ["./pg_migrate_tool", "-port", "8765"]

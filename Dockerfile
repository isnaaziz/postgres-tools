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

# Install PostgreSQL client tools (v17 for universal compatibility with v15, v16, v17)
RUN apt-get update && apt-get install -y curl gnupg2 lsb-release \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg \
    && echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update && apt-get install -y \
    postgresql-client-17 \
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

#!/bin/bash
# install.sh — setup dan build pg_migrate_tool

set -e

echo "🐘 Setup pg_migrate_tool..."

# Download dependencies
go mod tidy
go mod download

echo "📦 Build binary..."
go build -o pg_migrate_tool .

echo ""
echo "✅ Build selesai!"
echo ""
echo "Cara pakai:"
echo "  ./pg_migrate_tool                   # jalankan di port 8765 (default)"
echo "  ./pg_migrate_tool -port 9000        # port custom"
echo ""
echo "Buka browser: http://localhost:8765"

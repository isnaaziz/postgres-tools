# 🐘 pg_migrate_tool

Tools backup & restore PostgreSQL lintas versi dengan UI web berbasis Go + HTML.

## Fitur

- ✅ Backup **full database** (pg_dump format custom/plain/directory)
- ✅ Backup **per schema** (`-n schema_name`)
- ✅ Backup & restore **lintas versi** (PG 14/15/16/17 dst)
- ✅ Dukungan penuh **TimescaleDB** (pre/post restore + recompress chunks)
- ✅ **Parallel restore** (`-j N`)
- ✅ **Realtime log** via polling HTTP
- ✅ **Riwayat jobs** tersimpan in-memory selama server aktif
- ✅ Deteksi otomatis schema & hypertable
- ✅ Test koneksi sebelum backup/restore

## Persyaratan

- Go 1.22+
- `pg_dump`, `pg_restore`, `psql` (dari paket `postgresql-client`)

```bash
# Ubuntu/Debian
sudo apt install postgresql-client-common postgresql-client-17

# macOS
brew install libpq
export PATH="/opt/homebrew/opt/libpq/bin:$PATH"
```

## Instalasi & Build

```bash
git clone https://github.com/youruser/pg_migrate_tool
cd pg_migrate_tool
go mod tidy
go build -o pg_migrate_tool .
```

## Menjalankan

```bash
./pg_migrate_tool              # port default 8765
./pg_migrate_tool -port 9000   # port custom
```

Buka browser: **http://localhost:8765**

## Alur Backup Full

1. Isi host, port, user, password, nama database
2. Klik **Test Koneksi** untuk verifikasi
3. Biarkan schema kosong = full backup
4. Format `custom` = binary terkompresi (direkomendasikan)
5. Klik **Mulai Backup**

## Alur Backup Per-Schema

1. Klik **Load Schema** → klik schema yang diinginkan
2. Atau isi manual di field Schema (bisa lebih dari satu, pisah koma)
3. Tool menjalankan `pg_dump -n nama_schema`

## Alur Backup/Restore TimescaleDB

### Backup
```
pg_dump (schema only) → pg_dump (data, exclude _timescaledb_catalog) → simpan metadata
```

### Restore
```
CREATE EXTENSION timescaledb
  → timescaledb_pre_restore()
  → pg_restore
  → timescaledb_post_restore()
  → compress_chunk() untuk chunk yang perlu kompresi
```

## Restore Lintas Versi (PG 16 → PG 17)

`pg_dump` dan `pg_restore` mendukung migrasi lintas versi secara native. Tools ini menambahkan:

- `--no-owner` — skip error perbedaan ownership
- `--no-acl` — skip permission yang berbeda antar environment
- Log otomatis jika versi source ≠ target
- Buat database target otomatis jika belum ada

## Struktur Proyek

```
pg_migrate_tool/
├── main.go                          # Entry point, HTTP server
├── internal/
│   ├── api/handlers.go              # REST API handlers
│   ├── backup/backup.go             # pg_dump wrapper
│   ├── restore/restore.go           # pg_restore wrapper + TimescaleDB
│   ├── db/db.go                     # test conn, list schemas, detect timescale
│   └── jobs/store.go                # in-memory job registry
├── web/static/
│   └── index.html                   # Single-page UI
└── go.mod
```

## API Endpoints

| Method | Path | Deskripsi |
|--------|------|-----------|
| POST | `/api/test-conn` | Test koneksi PostgreSQL |
| POST | `/api/list-schemas` | List schema + hypertable |
| POST | `/api/backup` | Mulai backup (async, return job_id) |
| POST | `/api/restore` | Mulai restore (async, return job_id) |
| GET | `/api/jobs` | Daftar semua jobs |
| GET | `/api/job/{id}` | Detail + log satu job |

## Lisensi

MIT

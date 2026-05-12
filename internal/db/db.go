package db

import (
	"fmt"
	"os/exec"
	"strings"
)

// TestConn mencoba koneksi ke PostgreSQL, mengembalikan versi jika berhasil
func TestConn(host, port, user, password, dbname string) (string, error) {
	env := []string{"PGPASSWORD=" + password}

	target := "postgres"
	if dbname != "" {
		target = dbname
	}

	cmd := exec.Command("psql",
		"-h", host, "-p", port, "-U", user, "-d", target,
		"-t", "-c", "SELECT version();",
	)
	cmd.Env = append(cmd.Environ(), env...)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("koneksi gagal: %w", err)
	}

	version := strings.TrimSpace(string(out))
	// ambil bagian singkat: "PostgreSQL 16.3 ..."
	if idx := strings.Index(version, " on "); idx != -1 {
		version = version[:idx]
	}
	return version, nil
}

// ListSchemas mengembalikan daftar schema user (bukan system schema)
func ListSchemas(host, port, user, password, dbname string) ([]string, error) {
	env := []string{"PGPASSWORD=" + password}

	cmd := exec.Command("psql",
		"-h", host, "-p", port, "-U", user, "-d", dbname,
		"-t", "-c",
		`SELECT schema_name
		 FROM information_schema.schemata
		 WHERE schema_name NOT IN ('pg_catalog','information_schema','pg_toast')
		   AND schema_name NOT LIKE 'pg_temp%'
		   AND schema_name NOT LIKE 'pg_toast_temp%'
		 ORDER BY schema_name;`,
	)
	cmd.Env = append(cmd.Environ(), env...)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gagal list schema: %w", err)
	}

	var schemas []string
	for _, line := range strings.Split(string(out), "\n") {
		s := strings.TrimSpace(line)
		if s != "" {
			schemas = append(schemas, s)
		}
	}
	return schemas, nil
}

// DetectTimescale mengecek apakah database menggunakan TimescaleDB
func DetectTimescale(host, port, user, password, dbname string) (bool, string, error) {
	env := []string{"PGPASSWORD=" + password}

	cmd := exec.Command("psql",
		"-h", host, "-p", port, "-U", user, "-d", dbname,
		"-t", "-c",
		"SELECT extversion FROM pg_extension WHERE extname='timescaledb';",
	)
	cmd.Env = append(cmd.Environ(), env...)

	out, err := cmd.Output()
	if err != nil {
		return false, "", nil
	}
	ver := strings.TrimSpace(string(out))
	if ver == "" {
		return false, "", nil
	}
	return true, ver, nil
}

// ListHypertables mengembalikan daftar hypertable TimescaleDB
func ListHypertables(host, port, user, password, dbname string) ([]string, error) {
	env := []string{"PGPASSWORD=" + password}

	cmd := exec.Command("psql",
		"-h", host, "-p", port, "-U", user, "-d", dbname,
		"-t", "-c",
		"SELECT schema_name || '.' || table_name FROM _timescaledb_catalog.hypertable ORDER BY table_name;",
	)
	cmd.Env = append(cmd.Environ(), env...)

	out, err := cmd.Output()
	if err != nil {
		return nil, nil // mungkin bukan timescaledb, aman diabaikan
	}

	var tables []string
	for _, line := range strings.Split(string(out), "\n") {
		s := strings.TrimSpace(line)
		if s != "" {
			tables = append(tables, s)
		}
	}
	return tables, nil
}

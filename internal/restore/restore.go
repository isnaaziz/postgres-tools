package restore

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yourusername/pg_migrate_tool/internal/models"
)

// Run menjalankan restore berdasarkan config
func Run(cfg models.RestoreConfig, job *models.Job) error {
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}

	// Step 1: Validasi file
	job.AddLog("info", fmt.Sprintf("Memvalidasi file: %s", cfg.File))
	if _, err := os.Stat(cfg.File); os.IsNotExist(err) {
		return fmt.Errorf("file backup tidak ditemukan: %s", cfg.File)
	}
	job.AddLog("success", "File backup valid")

	// Step 2: Cek versi target PG
	pgVer, err := getPGVersion(cfg.Host, cfg.Port, cfg.User, env)
	if err != nil {
		job.AddLog("warn", "Tidak bisa membaca versi PostgreSQL target: "+err.Error())
	} else {
		job.AddLog("info", fmt.Sprintf("Target PostgreSQL: %s", pgVer))
		if cfg.SrcVersion != "" {
			job.AddLog("info", fmt.Sprintf("Source PostgreSQL: %s", cfg.SrcVersion))
			if cfg.SrcVersion != pgVer {
				job.AddLog("warn", fmt.Sprintf("Versi berbeda (%s → %s) — akan lanjut restore lintas versi", cfg.SrcVersion, pgVer))
			}
		}
	}

	// Step 3: Buat DB jika diminta
	if cfg.CreateDB {
		if err := createDatabase(cfg, env, job); err != nil {
			return err
		}
	}

	// Step 4: Pre-restore untuk TimescaleDB
	if cfg.Timescale {
		if err := timescalePreRestore(cfg, env, job); err != nil {
			return err
		}
	}

	// Step 5: Restore
	job.AddLog("info", "Memulai pg_restore...")
	if err := runPgRestore(cfg, env, job); err != nil {
		return err
	}

	// Step 6: Post-restore untuk TimescaleDB
	if cfg.Timescale {
		if err := timescalePostRestore(cfg, env, job); err != nil {
			return err
		}
	}

	job.AddLog("success", "Seluruh proses restore selesai dengan sukses")
	return nil
}

func runPgRestore(cfg models.RestoreConfig, env []string, job *models.Job) error {
	args := []string{
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-d", cfg.DB,
		"-v",
		"--no-owner", // agar tidak gagal karena beda owner
		"--no-acl",   // skip permission jika lintas env
		// "--exit-on-error", // Dihapus agar tidak gagal saat ada perbedaan parameter config (seperti transaction_timeout di PG17)
	}

	if cfg.Jobs > 1 {
		args = append(args, "-j", fmt.Sprintf("%d", cfg.Jobs))
	}

	if cfg.Schema != "" {
		args = append(args, "-n", cfg.Schema)
	}

	args = append(args, cfg.File)

	cmd := exec.Command("pg_restore", args...)
	cmd.Env = env

	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("pg_restore tidak ditemukan: %w", err)
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		level := "info"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") {
			level = "error"
		} else if strings.Contains(lower, "warn") {
			level = "warn"
		} else if strings.Contains(lower, "creating") || strings.Contains(lower, "processing") {
			level = "info"
		}
		job.AddLog(level, line)
	}
	if err := scanner.Err(); err != nil {
		job.AddLog("error", "Scanner error: "+err.Error())
	}

	if err := cmd.Wait(); err != nil {
		job.AddLog("warn", fmt.Sprintf("pg_restore selesai dengan peringatan: %v", err))
	} else {
		job.AddLog("success", "pg_restore berhasil diselesaikan")
	}

	return nil
}

func createDatabase(cfg models.RestoreConfig, env []string, job *models.Job) error {
	job.AddLog("info", fmt.Sprintf("Membuat database baru: %s", cfg.DB))

	// Cek apakah DB sudah ada
	checkCmd := exec.Command("psql",
		"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User,
		"-d", "postgres",
		"-tc", fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", cfg.DB),
	)
	checkCmd.Env = env
	out, _ := checkCmd.Output()

	if strings.TrimSpace(string(out)) == "1" {
		job.AddLog("warn", fmt.Sprintf("Database '%s' sudah ada, skip create", cfg.DB))
		return nil
	}

	// Install ekstensi TimescaleDB jika diperlukan
	createSQL := fmt.Sprintf("CREATE DATABASE \"%s\"", cfg.DB)
	createCmd := exec.Command("psql",
		"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User,
		"-d", "postgres",
		"-c", createSQL,
	)
	createCmd.Env = env

	if out, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gagal membuat database: %s — %w", string(out), err)
	}
	job.AddLog("success", fmt.Sprintf("Database '%s' berhasil dibuat", cfg.DB))
	return nil
}

func timescalePreRestore(cfg models.RestoreConfig, env []string, job *models.Job) error {
	job.AddLog("info", "TimescaleDB: menjalankan timescaledb_pre_restore()")

	// Pastikan ekstensi timescaledb sudah ada di target
	installCmd := exec.Command("psql",
		"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB,
		"-c", "CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;",
	)
	installCmd.Env = env
	if out, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gagal install timescaledb extension: %s — %w", string(out), err)
	}
	job.AddLog("success", "Ekstensi timescaledb aktif")

	// Jalankan pre-restore
	preCmd := exec.Command("psql",
		"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB,
		"-c", "SELECT timescaledb_pre_restore();",
	)
	preCmd.Env = env
	if out, err := preCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("timescaledb_pre_restore() gagal: %s — %w", string(out), err)
	}
	job.AddLog("success", "timescaledb_pre_restore() selesai")
	return nil
}

func timescalePostRestore(cfg models.RestoreConfig, env []string, job *models.Job) error {
	job.AddLog("info", "TimescaleDB: menjalankan timescaledb_post_restore()")

	postCmd := exec.Command("psql",
		"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB,
		"-c", "SELECT timescaledb_post_restore();",
	)
	postCmd.Env = env
	if out, err := postCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("timescaledb_post_restore() gagal: %s — %w", string(out), err)
	}
	job.AddLog("success", "timescaledb_post_restore() selesai")

	// Rekompresi chunk
	job.AddLog("info", "Mengkompresi ulang hypertable chunks...")
	compressCmd := exec.Command("psql",
		"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB,
		"-c", `
			SELECT compress_chunk(c.schema_name || '.' || c.table_name)
			FROM _timescaledb_catalog.chunk c
			JOIN _timescaledb_catalog.hypertable h ON h.id = c.hypertable_id
			WHERE h.compression_state = 1
			  AND c.compressed_chunk_id IS NULL
			LIMIT 50;
		`,
	)
	compressCmd.Env = env
	if out, err := compressCmd.CombinedOutput(); err != nil {
		job.AddLog("warn", fmt.Sprintf("Kompresi chunk warning (opsional): %s", string(out)))
	} else {
		job.AddLog("success", "Kompresi chunk selesai")
	}

	return nil
}

func getPGVersion(host, port, user string, env []string) (string, error) {
	cmd := exec.Command("psql",
		"-h", host, "-p", port, "-U", user, "-d", "postgres",
		"-t", "-c", "SHOW server_version;",
	)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

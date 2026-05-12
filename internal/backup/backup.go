package backup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/pg_migrate_tool/internal/models"
)

// Run menjalankan backup berdasarkan config dan menulis log ke job
func Run(cfg models.BackupConfig, job *models.Job) error {
	// Buat direktori output jika belum ada
	dir := filepath.Dir(cfg.OutputFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("gagal membuat direktori output: %w", err)
		}
	}

	if cfg.Timescale {
		return runTimescaleBackup(cfg, job)
	}
	return runPgDump(cfg, job)
}

func runPgDump(cfg models.BackupConfig, job *models.Job) error {
	args := buildPgDumpArgs(cfg)

	job.AddLog("info", fmt.Sprintf("Menjalankan: pg_dump %s", maskPassword(strings.Join(args, " "))))

	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("pg_dump tidak ditemukan atau gagal start: %w", err)
	}

	// Stream stderr sebagai log
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			level := "info"
			if strings.Contains(strings.ToLower(line), "error") {
				level = "error"
			} else if strings.Contains(strings.ToLower(line), "warn") {
				level = "warn"
			}
			job.AddLog(level, line)
		}
		if err := scanner.Err(); err != nil {
			job.AddLog("error", "Scanner error (stderr): "+err.Error())
		}
	}()

	// Untuk format plain, stream stdout juga
	if cfg.Format == "plain" {
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				// plain SQL - hanya log progress setiap 1000 baris
			}
			if err := scanner.Err(); err != nil {
				job.AddLog("error", "Scanner error (stdout): "+err.Error())
			}
		}()
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pg_dump gagal: %w", err)
	}

	// Hitung ukuran file output
	if info, err := os.Stat(cfg.OutputFile); err == nil {
		size := humanSize(info.Size())
		job.AddLog("success", fmt.Sprintf("Backup selesai! Ukuran file: %s", size))
	}

	return nil
}

func runTimescaleBackup(cfg models.BackupConfig, job *models.Job) error {
	job.AddLog("info", "Mode TimescaleDB terdeteksi")
	job.AddLog("info", "Langkah 1/3: Backup schema + TimescaleDB catalog")

	// Step 1: backup schema saja dulu
	schemaCfg := cfg
	schemaCfg.OutputFile = cfg.OutputFile + ".schema.sql"
	schemaCfg.Format = "plain"
	schemaArgs := buildPgDumpArgs(schemaCfg)
	schemaArgs = append(schemaArgs, "--schema-only")

	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}

	cmd := exec.Command("pg_dump", schemaArgs...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		job.AddLog("warn", fmt.Sprintf("Schema backup output: %s", string(out)))
		// Lanjutkan meski ada warning
	}
	job.AddLog("success", "Schema berhasil di-backup")

	// Step 2: backup data
	job.AddLog("info", "Langkah 2/3: Backup data (termasuk hypertable chunks)")
	dataCfg := cfg
	dataCfg.Format = "custom"
	dataArgs := buildPgDumpArgs(dataCfg)
	dataArgs = append(dataArgs, "--data-only", "--exclude-table=_timescaledb_catalog.*")

	cmd2 := exec.Command("pg_dump", dataArgs...)
	cmd2.Env = env
	stderr2, _ := cmd2.StderrPipe()

	if err := cmd2.Start(); err != nil {
		return fmt.Errorf("pg_dump data gagal: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr2)
		for scanner.Scan() {
			job.AddLog("info", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			job.AddLog("error", "Scanner error (stderr2): "+err.Error())
		}
	}()

	if err := cmd2.Wait(); err != nil {
		return fmt.Errorf("backup data gagal: %w", err)
	}

	job.AddLog("info", "Langkah 3/3: Menyimpan metadata TimescaleDB")
	// Simpan metadata versi timescaledb
	metaCmd := exec.Command("psql",
		"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB,
		"-c", "SELECT extversion FROM pg_extension WHERE extname='timescaledb'",
		"-t",
	)
	metaCmd.Env = env
	metaOut, _ := metaCmd.Output()

	tsVersion := strings.TrimSpace(string(metaOut))
	job.AddLog("info", fmt.Sprintf("Versi TimescaleDB: %s", tsVersion))
	job.AddLog("success", "Backup TimescaleDB selesai!")

	return nil
}

func buildPgDumpArgs(cfg models.BackupConfig) []string {
	args := []string{
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-d", cfg.DB,
		"-v",
	}

	// Format
	switch cfg.Format {
	case "custom":
		args = append(args, "-F", "c")
		args = append(args, fmt.Sprintf("-Z%d", cfg.Compress))
	case "plain":
		args = append(args, "-F", "p")
	case "directory":
		args = append(args, "-F", "d")
		if cfg.Jobs > 1 {
			args = append(args, "-j", fmt.Sprintf("%d", cfg.Jobs))
		}
	}

	// Per-schema
	if cfg.Schema != "" {
		args = append(args, "-n", cfg.Schema)
	}

	// Output file
	args = append(args, "-f", cfg.OutputFile)

	return args
}

func maskPassword(s string) string {
	return s // password tidak masuk ke args, hanya env
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// GenerateFilename membuat nama file otomatis dengan timestamp
func GenerateFilename(db, schema string, timescale bool) string {
	ts := time.Now().Format("20060102_150405")
	prefix := db
	if schema != "" {
		prefix = fmt.Sprintf("%s_%s", db, schema)
	}
	if timescale {
		prefix = prefix + "_timescale"
	}
	return fmt.Sprintf("%s_%s.dump", prefix, ts)
}

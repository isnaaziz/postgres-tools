package restore

import (
	"fmt"
	"os/exec"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func timescalePreRestore(cfg models.RestoreConfig, env []string, job *models.Job) error {
	job.AddLog("info", "TimescaleDB: menjalankan timescaledb_pre_restore()")

	installCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-c", "CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;")
	installCmd.Env = env
	if out, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gagal install timescaledb: %s — %w", string(out), err)
	}
	job.AddLog("success", "Ekstensi timescaledb aktif")

	preCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-c", "SELECT timescaledb_pre_restore();")
	preCmd.Env = env
	if out, err := preCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("timescaledb_pre_restore() gagal: %s — %w", string(out), err)
	}
	job.AddLog("success", "timescaledb_pre_restore() selesai")
	return nil
}

func timescalePostRestore(cfg models.RestoreConfig, env []string, job *models.Job) error {
	job.AddLog("info", "TimescaleDB: menjalankan timescaledb_post_restore()")

	postCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-c", "SELECT timescaledb_post_restore();")
	postCmd.Env = env
	if out, err := postCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("timescaledb_post_restore() gagal: %s — %w", string(out), err)
	}
	job.AddLog("success", "timescaledb_post_restore() selesai")

	job.AddLog("info", "Mengkompresi ulang hypertable chunks...")
	compressCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-c", "SELECT compress_chunk(c.schema_name || '.' || c.table_name) FROM _timescaledb_catalog.chunk c JOIN _timescaledb_catalog.hypertable h ON h.id = c.hypertable_id WHERE h.compression_state = 1;")
	compressCmd.Env = env
	compressCmd.Run()

	return nil
}

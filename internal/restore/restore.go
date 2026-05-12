package restore

import (
	"fmt"
	"os"
	"os/exec"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func Run(cfg models.RestoreConfig, job *models.Job) error {
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}

	pgVer, err := getPgVersion(cfg, env)
	if err != nil {
		job.AddLog("warn", "Tidak bisa membaca versi PostgreSQL target: "+err.Error())
	} else {
		job.AddLog("info", fmt.Sprintf("Target PostgreSQL: %s", pgVer))
		if cfg.SrcVersion != "" {
			job.AddLog("info", fmt.Sprintf("Source PostgreSQL: %s", cfg.SrcVersion))
		}
	}

	if cfg.CreateDB {
		if err := createDatabase(cfg, env, job); err != nil {
			return err
		}
	}

	if cfg.Schema != "" {
		job.AddLog("info", fmt.Sprintf("Memastikan schema '%s' tersedia...", cfg.Schema))
		createSchemaCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-c", fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS \"%s\";", cfg.Schema))
		createSchemaCmd.Env = env
		createSchemaCmd.Run()
	}

	if cfg.Timescale {
		if err := timescalePreRestore(cfg, env, job); err != nil {
			return err
		}

		schemaFile := cfg.File + ".schema.sql"
		if _, err := os.Stat(schemaFile); err == nil {
			job.AddLog("info", "TimescaleDB: merestore struktur dari .schema.sql...")
			restoreSchemaCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-f", schemaFile)
			restoreSchemaCmd.Env = env
			if out, err := restoreSchemaCmd.CombinedOutput(); err != nil {
				job.AddLog("warn", "Log restore schema: "+string(out))
			} else {
				job.AddLog("success", "Struktur tabel berhasil dipulihkan")
			}
		}
	}

	job.AddLog("info", "Memulai pg_restore...")
	if err := runPgRestore(cfg, env, job); err != nil {
		return err
	}

	if cfg.Timescale {
		if err := timescalePostRestore(cfg, env, job); err != nil {
			return err
		}
	}

	job.AddLog("success", "Seluruh proses restore selesai dengan sukses")
	return nil
}

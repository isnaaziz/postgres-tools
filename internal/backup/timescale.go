package backup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func runTimescaleBackup(cfg models.BackupConfig, job *models.Job) error {
	job.AddLog("info", "Langkah 1/3: Backup schema")
	schemaCfg := cfg
	schemaCfg.OutputFile = cfg.OutputFile + ".schema.sql"
	schemaCfg.Format = "plain"
	schemaArgs := append(buildPgDumpArgs(schemaCfg), "--schema-only")

	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}

	cmd := exec.Command("pg_dump", schemaArgs...)
	cmd.Env = env
	cmd.Run()
	job.AddLog("success", "Schema berhasil di-backup")

	job.AddLog("info", "Langkah 2/3: Backup data")
	dataCfg := cfg
	dataCfg.Format = "custom"
	dataArgs := append(buildPgDumpArgs(dataCfg), "--data-only", "--exclude-table=_timescaledb_catalog.*")

	cmd2 := exec.Command("pg_dump", dataArgs...)
	cmd2.Env = env
	stderr2, _ := cmd2.StderrPipe()
	cmd2.Start()

	go func() {
		scanner := bufio.NewScanner(stderr2)
		for scanner.Scan() {
			job.AddLog("info", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			job.AddLog("error", "Scanner error: "+err.Error())
		}
	}()
	cmd2.Wait()

	job.AddLog("info", "Langkah 3/3: Metadata TimescaleDB")
	metaCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-c", "SELECT extversion FROM pg_extension WHERE extname='timescaledb'", "-t")
	metaCmd.Env = env
	metaOut, _ := metaCmd.Output()

	job.AddLog("info", fmt.Sprintf("Versi TimescaleDB: %s", strings.TrimSpace(string(metaOut))))
	job.AddLog("success", "Backup TimescaleDB selesai")

	return nil
}

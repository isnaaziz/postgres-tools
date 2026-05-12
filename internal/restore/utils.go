package restore

import (
	"fmt"
	"os/exec"
	"strings"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func createDatabase(cfg models.RestoreConfig, env []string, job *models.Job) error {
	job.AddLog("info", fmt.Sprintf("Membuat database baru: %s", cfg.DB))

	checkCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", "postgres", "-tc", fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", cfg.DB))
	checkCmd.Env = env
	out, _ := checkCmd.Output()

	if strings.TrimSpace(string(out)) == "1" {
		job.AddLog("warn", fmt.Sprintf("Database '%s' sudah ada, skip create", cfg.DB))
		return nil
	}

	createCmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", "postgres", "-c", fmt.Sprintf("CREATE DATABASE \"%s\"", cfg.DB))
	createCmd.Env = env
	if out, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gagal membuat database: %s — %w", string(out), err)
	}
	job.AddLog("success", fmt.Sprintf("Database '%s' berhasil dibuat", cfg.DB))
	return nil
}

func getPgVersion(cfg models.RestoreConfig, env []string) (string, error) {
	cmd := exec.Command("psql", "-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-tc", "SHOW server_version")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	ver := strings.TrimSpace(string(out))
	if idx := strings.Index(ver, " "); idx != -1 {
		ver = ver[:idx]
	}
	return ver, nil
}

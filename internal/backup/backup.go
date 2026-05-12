package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func Run(cfg models.BackupConfig, job *models.Job) error {
	dir := filepath.Dir(cfg.OutputFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("gagal membuat direktori: %w", err)
		}
	}

	if cfg.Timescale {
		return runTimescaleBackup(cfg, job)
	}
	return runPgDump(cfg, job)
}

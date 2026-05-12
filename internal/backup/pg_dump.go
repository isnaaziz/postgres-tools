package backup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func runPgDump(cfg models.BackupConfig, job *models.Job) error {
	args := buildPgDumpArgs(cfg)
	env := os.Environ()
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = env

	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("pg_dump gagal: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			level := "info"
			lower := strings.ToLower(line)
			if strings.Contains(lower, "error") {
				level = "error"
			} else if strings.Contains(lower, "warn") {
				level = "warn"
			}
			job.AddLog(level, line)
		}
		if err := scanner.Err(); err != nil {
			job.AddLog("error", "Scanner error: "+err.Error())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pg_dump gagal: %w", err)
	}

	if info, err := os.Stat(cfg.OutputFile); err == nil {
		job.AddLog("success", fmt.Sprintf("Backup selesai! Ukuran file: %s", humanSize(info.Size())))
	}

	return nil
}

func buildPgDumpArgs(cfg models.BackupConfig) []string {
	args := []string{"-h", cfg.Host, "-p", cfg.Port, "-U", cfg.User, "-d", cfg.DB, "-v"}

	switch cfg.Format {
	case "custom":
		args = append(args, "-F", "c", fmt.Sprintf("-Z%d", cfg.Compress))
	case "plain":
		args = append(args, "-F", "p")
	case "directory":
		args = append(args, "-F", "d")
		if cfg.Jobs > 1 {
			args = append(args, "-j", fmt.Sprintf("%d", cfg.Jobs))
		}
	}

	if cfg.Schema != "" {
		args = append(args, "-n", cfg.Schema)
	}

	args = append(args, "-f", cfg.OutputFile)
	return args
}

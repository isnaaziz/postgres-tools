package restore

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func runPgRestore(cfg models.RestoreConfig, env []string, job *models.Job) error {
	args := []string{
		"-h", cfg.Host,
		"-p", cfg.Port,
		"-U", cfg.User,
		"-d", cfg.DB,
		"-v",
		"--no-owner",
		"--no-acl",
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
		return fmt.Errorf("pg_restore gagal: %w", err)
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

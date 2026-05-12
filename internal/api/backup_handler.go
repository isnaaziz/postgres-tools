package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"github.com/yourusername/pg_migrate_tool/internal/backup"
	"github.com/yourusername/pg_migrate_tool/internal/db"
	"github.com/yourusername/pg_migrate_tool/internal/jobs"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func HandleTestConn(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions { return }
	var req models.TestConnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, "request tidak valid")
		return
	}
	version, err := db.TestConn(req.Host, req.Port, req.User, req.Password, req.DB)
	if err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	hasTS, tsVer, _ := db.DetectTimescale(req.Host, req.Port, req.User, req.Password, req.DB)
	json200(w, map[string]any{"ok": true, "version": version, "has_timescale": hasTS, "timescale_ver": tsVer})
}

func HandleListSchemas(w http.ResponseWriter, r *http.Request) {
	cors(w)
	var req models.ListSchemasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, "request tidak valid")
		return
	}
	schemas, err := db.ListSchemas(req.Host, req.Port, req.User, req.Password, req.DB)
	if err != nil {
		jsonErr(w, 502, err.Error())
		return
	}
	hypertables, _ := db.ListHypertables(req.Host, req.Port, req.User, req.Password, req.DB)
	json200(w, map[string]any{"schemas": schemas, "hypertables": hypertables})
}

func HandleBackup(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions { return }
	var req models.BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, "request tidak valid")
		return
	}
	if req.DB == "" {
		jsonErr(w, 400, "nama database wajib diisi")
		return
	}
	if req.Port == "" { req.Port = "5432" }
	if req.Format == "" { req.Format = "custom" }
	if req.Jobs == 0 { req.Jobs = 4 }
	if req.Compress == 0 { req.Compress = 6 }
	if req.OutputFile == "" {
		fname := backup.GenerateFilename(req.DB, req.Schema, req.Timescale)
		req.OutputFile = filepath.Join("backups", fname)
	}
	job := jobs.Global.New("backup", req.DB, req.Schema, req.OutputFile)
	go func() {
		cfg := models.BackupConfig{
			Host: req.Host, Port: req.Port, User: req.User, Password: req.Password, DB: req.DB,
			Schema: req.Schema, OutputFile: req.OutputFile, Format: req.Format, Jobs: req.Jobs,
			Compress: req.Compress, Timescale: req.Timescale,
		}
		job.AddLog("info", fmt.Sprintf("Backup dimulai: db=%s schema=%s file=%s", req.DB, req.Schema, req.OutputFile))
		if err := backup.Run(cfg, job); err != nil {
			job.AddLog("error", "Backup GAGAL: "+err.Error())
			job.Finish(false)
		} else {
			job.AddLog("success", "Job Backup selesai: File berhasil dibuat")
			job.Finish(true)
		}
	}()
	json200(w, map[string]string{"job_id": job.ID})
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/yourusername/pg_migrate_tool/internal/jobs"
	"github.com/yourusername/pg_migrate_tool/internal/models"
	"github.com/yourusername/pg_migrate_tool/internal/restore"
)

func HandleRestore(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions { return }
	var req models.RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, "request tidak valid")
		return
	}
	if req.DB == "" || req.File == "" {
		jsonErr(w, 400, "db dan file wajib diisi")
		return
	}
	if req.Port == "" { req.Port = "5432" }
	if req.Jobs == 0 { req.Jobs = 4 }
	job := jobs.Global.New("restore", req.DB, req.Schema, req.File)
	go func() {
		cfg := models.RestoreConfig{
			Host: req.Host, Port: req.Port, User: req.User, Password: req.Password, DB: req.DB,
			Schema: req.Schema, File: req.File, Jobs: req.Jobs, Timescale: req.Timescale,
			CreateDB: req.CreateDB, SrcVersion: req.SrcVersion,
		}
		job.AddLog("info", fmt.Sprintf("Restore dimulai: file=%s → db=%s", req.File, req.DB))
		if err := restore.Run(cfg, job); err != nil {
			job.AddLog("error", "Restore GAGAL: "+err.Error())
			job.Finish(false)
		} else {
			job.AddLog("success", "Job Restore selesai: Database telah dipulihkan")
			job.Finish(true)
		}
	}()
	json200(w, map[string]string{"job_id": job.ID})
}

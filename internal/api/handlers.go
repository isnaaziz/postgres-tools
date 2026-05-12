package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/yourusername/pg_migrate_tool/internal/backup"
	"github.com/yourusername/pg_migrate_tool/internal/db"
	"github.com/yourusername/pg_migrate_tool/internal/jobs"
	"github.com/yourusername/pg_migrate_tool/internal/models"
	"github.com/yourusername/pg_migrate_tool/internal/restore"
)

func json200(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// POST /api/test-conn
func HandleTestConn(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
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

	json200(w, map[string]any{
		"ok":             true,
		"version":        version,
		"has_timescale":  hasTS,
		"timescale_ver":  tsVer,
	})
}

// POST /api/list-schemas
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

	json200(w, map[string]any{
		"schemas":     schemas,
		"hypertables": hypertables,
	})
}

// POST /api/backup — async, langsung return job ID
func HandleBackup(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req models.BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, "request tidak valid")
		return
	}

	if req.DB == "" {
		jsonErr(w, 400, "nama database wajib diisi")
		return
	}
	if req.Port == "" {
		req.Port = "5432"
	}
	if req.Format == "" {
		req.Format = "custom"
	}
	if req.Jobs == 0 {
		req.Jobs = 4
	}
	if req.Compress == 0 {
		req.Compress = 6
	}
	if req.OutputFile == "" {
		req.OutputFile = backup.GenerateFilename(req.DB, req.Schema, req.Timescale)
	}

	job := jobs.Global.New("backup", req.DB, req.Schema, req.OutputFile)

	go func() {
		cfg := models.BackupConfig{
			Host:       req.Host,
			Port:       req.Port,
			User:       req.User,
			Password:   req.Password,
			DB:         req.DB,
			Schema:     req.Schema,
			OutputFile: req.OutputFile,
			Format:     req.Format,
			Jobs:       req.Jobs,
			Compress:   req.Compress,
			Timescale:  req.Timescale,
		}
		job.AddLog("info", fmt.Sprintf("Backup dimulai: db=%s schema=%s file=%s", req.DB, req.Schema, req.OutputFile))
		if err := backup.Run(cfg, job); err != nil {
			job.AddLog("error", err.Error())
			job.Finish(false)
		} else {
			job.Finish(true)
		}
	}()

	json200(w, map[string]string{"job_id": job.ID})
}

// POST /api/restore — async
func HandleRestore(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req models.RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, "request tidak valid")
		return
	}

	if req.DB == "" || req.File == "" {
		jsonErr(w, 400, "db dan file wajib diisi")
		return
	}
	if req.Port == "" {
		req.Port = "5432"
	}
	if req.Jobs == 0 {
		req.Jobs = 4
	}

	job := jobs.Global.New("restore", req.DB, req.Schema, req.File)

	go func() {
		cfg := models.RestoreConfig{
			Host:       req.Host,
			Port:       req.Port,
			User:       req.User,
			Password:   req.Password,
			DB:         req.DB,
			Schema:     req.Schema,
			File:       req.File,
			Jobs:       req.Jobs,
			Timescale:  req.Timescale,
			CreateDB:   req.CreateDB,
			SrcVersion: req.SrcVersion,
		}
		job.AddLog("info", fmt.Sprintf("Restore dimulai: file=%s → db=%s", req.File, req.DB))
		if err := restore.Run(cfg, job); err != nil {
			job.AddLog("error", err.Error())
			job.Finish(false)
		} else {
			job.Finish(true)
		}
	}()

	json200(w, map[string]string{"job_id": job.ID})
}

// GET /api/jobs
func HandleListJobs(w http.ResponseWriter, r *http.Request) {
	cors(w)
	list := jobs.Global.All()
	sort.Slice(list, func(i, j int) bool {
		return list[i].StartedAt.After(list[j].StartedAt)
	})
	json200(w, list)
}

// GET /api/job/{id}
func HandleJobStatus(w http.ResponseWriter, r *http.Request) {
	cors(w)
	id := strings.TrimPrefix(r.URL.Path, "/api/job/")
	job, ok := jobs.Global.Get(id)
	if !ok {
		jsonErr(w, 404, "job tidak ditemukan")
		return
	}
	json200(w, job)
}

// WebSocket /ws/logs?job_id=xxx — stream log realtime
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		conn.WriteJSON(map[string]string{"error": "job_id diperlukan"})
		return
	}

	var lastIdx int
	for {
		job, ok := jobs.Global.Get(jobID)
		if !ok {
			conn.WriteJSON(map[string]string{"error": "job tidak ditemukan"})
			return
		}

		if len(job.Logs) > lastIdx {
			for _, line := range job.Logs[lastIdx:] {
				if err := conn.WriteJSON(line); err != nil {
					return
				}
			}
			lastIdx = len(job.Logs)
		}

		if job.Status != models.StatusRunning {
			conn.WriteJSON(map[string]any{
				"done":   true,
				"status": job.Status,
			})
			return
		}

		// Poll setiap 200ms
		select {
		case <-r.Context().Done():
			return
		default:
			// sleep minimal tanpa import time package terpisah
		}
	}
}

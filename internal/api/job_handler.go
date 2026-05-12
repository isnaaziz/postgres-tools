package api

import (
	"net/http"
	"sort"
	"strings"
	"time"
	"github.com/gorilla/websocket"
	"github.com/yourusername/pg_migrate_tool/internal/jobs"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

func HandleListJobs(w http.ResponseWriter, r *http.Request) {
	cors(w)
	list := jobs.Global.All()
	sort.Slice(list, func(i, j int) bool { return list[i].StartedAt.After(list[j].StartedAt) })
	json200(w, list)
}

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

var upgrader = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	defer conn.Close()
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		conn.WriteJSON(map[string]string{"error": "job_id diperlukan"})
		return
	}
	var lastIdx int
	for {
		job, ok := jobs.Global.Get(jobID)
		if !ok { return }
		if len(job.Logs) > lastIdx {
			for _, line := range job.Logs[lastIdx:] {
				if err := conn.WriteJSON(line); err != nil { return }
			}
			lastIdx = len(job.Logs)
		}
		if job.Status != models.StatusRunning {
			conn.WriteJSON(map[string]any{"done": true, "status": job.Status})
			return
		}
		select {
		case <-r.Context().Done(): return
		case <-time.After(200 * time.Millisecond):
		}
	}
}

package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"github.com/yourusername/pg_migrate_tool/internal/api"
)

//go:embed all:web/static
var staticFiles embed.FS

func main() {
	port := flag.Int("port", 8765, "Port untuk web UI")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/backup", api.HandleBackup)
	mux.HandleFunc("/api/restore", api.HandleRestore)
	mux.HandleFunc("/api/list-schemas", api.HandleListSchemas)
	mux.HandleFunc("/api/test-conn", api.HandleTestConn)
	mux.HandleFunc("/api/jobs", api.HandleListJobs)
	mux.HandleFunc("/api/job/", api.HandleJobStatus)
	mux.HandleFunc("/api/list-files", api.HandleListFiles)
	mux.HandleFunc("/api/download", api.HandleDownload)
	mux.HandleFunc("/api/upload", api.HandleUpload)
	mux.HandleFunc("/ws/logs", api.HandleWebSocket)

	sub, err := fs.Sub(staticFiles, "web/static")
	if err != nil { log.Fatal(err) }
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("\n🐘 PG Migrate Tool berjalan di http://localhost%s\n\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

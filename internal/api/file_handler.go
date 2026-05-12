package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	cors(w)
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "file parameter diperlukan", 400)
		return
	}
	base := filepath.Base(file)
	path := filepath.Join("backups", base)
	if _, err := os.Stat(path); os.IsNotExist(err) { path = base }
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", base))
	http.ServeFile(w, r, path)
}

func HandleUpload(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions { return }
	r.ParseMultipartForm(2000 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonErr(w, 400, "gagal membaca file: "+err.Error())
		return
	}
	defer file.Close()
	os.MkdirAll("backups", 0755)
	dstPath := filepath.Join("backups", header.Filename)
	dst, _ := os.Create(dstPath)
	defer dst.Close()
	io.Copy(dst, file)
	json200(w, map[string]string{"message": "file berhasil diunggah", "path": dstPath})
}

func HandleListFiles(w http.ResponseWriter, r *http.Request) {
	cors(w)
	var files []map[string]any
	dirs := []string{".", "backups"}
	for _, dir := range dirs {
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			if entry.IsDir() { continue }
			name := entry.Name()
			ext := filepath.Ext(name)
			if ext == ".dump" || ext == ".sql" || ext == ".gz" {
				info, _ := entry.Info()
				path := name
				if dir != "." { path = filepath.Join(dir, name) }
				files = append(files, map[string]any{"name": name, "path": path, "size": info.Size(), "time": info.ModTime()})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i]["time"].(time.Time).After(files[j]["time"].(time.Time)) })
	json200(w, files)
}

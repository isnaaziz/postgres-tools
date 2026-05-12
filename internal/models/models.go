package models

import (
	"sync"
	"time"
)

// Status mewakili status dari sebuah job
type Status string

const (
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

// LogLine menyimpan satu baris log dalam job
type LogLine struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"` // info | warn | error | success
	Message string    `json:"message"`
}

// Job merepresentasikan tugas backup atau restore
type Job struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"` // backup | restore
	DB        string     `json:"db"`
	Schema    string     `json:"schema"`
	File      string     `json:"file"`
	Status    Status     `json:"status"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Logs      []LogLine  `json:"logs"`
	Mu        sync.Mutex `json:"-"`
}

func (j *Job) AddLog(level, msg string) {
	j.Mu.Lock()
	defer j.Mu.Unlock()
	j.Logs = append(j.Logs, LogLine{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
	})
}

func (j *Job) Finish(success bool) {
	j.Mu.Lock()
	defer j.Mu.Unlock()
	now := time.Now()
	j.EndedAt = &now
	if success {
		j.Status = StatusDone
	} else {
		j.Status = StatusFailed
	}
}

// BackupConfig konfigurasi untuk proses backup
type BackupConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	DB         string
	Schema     string // kosong = full backup
	OutputFile string
	Format     string // custom | plain | directory
	Jobs       int    // paralel jobs
	Compress   int    // 0-9
	Timescale  bool
}

// RestoreConfig konfigurasi untuk proses restore
type RestoreConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	DB         string
	Schema     string
	File       string
	Jobs       int
	Timescale  bool
	CreateDB   bool
	SrcVersion string // opsional, untuk logging
}

// --- API Request/Response Models ---

type TestConnRequest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DB       string `json:"db"`
}

type ListSchemasRequest struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DB       string `json:"db"`
}

type BackupRequest struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	DB         string `json:"db"`
	Schema     string `json:"schema"`
	OutputFile string `json:"output_file"`
	Format     string `json:"format"`
	Jobs       int    `json:"jobs"`
	Compress   int    `json:"compress"`
	Timescale  bool   `json:"timescale"`
}

type RestoreRequest struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	DB         string `json:"db"`
	Schema     string `json:"schema"`
	File       string `json:"file"`
	Jobs       int    `json:"jobs"`
	Timescale  bool   `json:"timescale"`
	CreateDB   bool   `json:"create_db"`
	SrcVersion string `json:"src_version"`
}

package jobs

import (
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/yourusername/pg_migrate_tool/internal/models"
)

type Store struct {
	mu   sync.RWMutex
	jobs map[string]*models.Job
}

var Global = &Store{jobs: make(map[string]*models.Job)}

func (s *Store) New(jobType, db, schema, file string) *models.Job {
	j := &models.Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		DB:        db,
		Schema:    schema,
		File:      file,
		Status:    models.StatusRunning,
		StartedAt: time.Now(),
		Logs:      []models.LogLine{},
	}
	s.mu.Lock()
	s.jobs[j.ID] = j
	s.mu.Unlock()
	return j
}

func (s *Store) Get(id string) (*models.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

func (s *Store) All() []*models.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*models.Job, 0, len(s.jobs))
	for _, j := range s.jobs { list = append(list, j) }
	return list
}

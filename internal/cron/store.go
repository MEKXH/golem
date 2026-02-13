package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// storeData is the on-disk format for cron jobs.
type storeData struct {
	Version int    `json:"version"`
	Jobs    []*Job `json:"jobs"`
}

// Store persists cron jobs as a JSON file.
type Store struct {
	path string
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewStore creates a store backed by the given file path.
func NewStore(path string) *Store {
	return &Store{
		path: path,
		jobs: make(map[string]*Job),
	}
}

// Load reads jobs from disk. If the file does not exist, the store starts empty.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.jobs = make(map[string]*Job)
			return nil
		}
		return fmt.Errorf("read cron store: %w", err)
	}

	var sd storeData
	if err := json.Unmarshal(data, &sd); err != nil {
		return fmt.Errorf("parse cron store: %w", err)
	}

	s.jobs = make(map[string]*Job, len(sd.Jobs))
	for _, j := range sd.Jobs {
		s.jobs[j.ID] = j
	}
	return nil
}

// Save writes all jobs to disk. Serialization happens under the read lock
// so that concurrent mutations via Put cannot race with encoding.
func (s *Store) Save() error {
	s.mu.RLock()
	sd := storeData{
		Version: 1,
		Jobs:    make([]*Job, 0, len(s.jobs)),
	}
	for _, j := range s.jobs {
		sd.Jobs = append(sd.Jobs, j)
	}
	data, err := json.MarshalIndent(sd, "", "  ")
	s.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshal cron store: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create cron store dir: %w", err)
	}

	return os.WriteFile(s.path, data, 0644)
}

// Put stores a deep copy of the job. The caller may continue to mutate the
// original without racing with Save or other readers.
func (s *Store) Put(job *Job) {
	cp := copyJob(job)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[cp.ID] = cp
}

// Get returns a deep copy of the job with the given ID.
func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	return copyJob(j), true
}

// Delete removes a job by ID.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[id]; !ok {
		return false
	}
	delete(s.jobs, id)
	return true
}

// All returns deep copies of all jobs.
func (s *Store) All() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		result = append(result, copyJob(j))
	}
	return result
}

// copyJob returns a deep copy of a Job, including all pointer fields.
func copyJob(j *Job) *Job {
	cp := *j
	if j.Schedule.AtMS != nil {
		v := *j.Schedule.AtMS
		cp.Schedule.AtMS = &v
	}
	if j.Schedule.EveryMS != nil {
		v := *j.Schedule.EveryMS
		cp.Schedule.EveryMS = &v
	}
	if j.State.NextRunAtMS != nil {
		v := *j.State.NextRunAtMS
		cp.State.NextRunAtMS = &v
	}
	if j.State.LastRunAtMS != nil {
		v := *j.State.LastRunAtMS
		cp.State.LastRunAtMS = &v
	}
	return &cp
}

package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// storeData 是定时任务在磁盘上的存储格式。
type storeData struct {
	Version int    `json:"version"`
	Jobs    []*Job `json:"jobs"`
}

// Store 将定时任务持久化为 JSON 文件。
type Store struct {
	path string
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewStore 创建一个由给定文件路径支持的存储。
func NewStore(path string) *Store {
	return &Store{
		path: path,
		jobs: make(map[string]*Job),
	}
}

// Load 从磁盘读取任务。如果文件不存在，存储将为空。
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

// Save 将所有任务写入磁盘。序列化在读锁下进行，
// 以便通过 Put 的并发修改不会与编码竞争。
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

// Put 存储任务的深拷贝。调用者可以继续修改原始任务，
// 而不会与 Save 或其他读取者竞争。
func (s *Store) Put(job *Job) {
	cp := copyJob(job)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[cp.ID] = cp
}

// Get 返回具有给定 ID 的任务的深拷贝。
func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	return copyJob(j), true
}

// Delete 按 ID 删除任务。
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[id]; !ok {
		return false
	}
	delete(s.jobs, id)
	return true
}

// All 返回所有任务的深拷贝。
func (s *Store) All() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		result = append(result, copyJob(j))
	}
	return result
}

// CopyJob 返回一个 Job 的深拷贝，包括所有指针字段。
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

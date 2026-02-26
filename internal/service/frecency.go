package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// frecencyRecord captures usage data for frecency scoring.
type frecencyRecord struct {
	ID       string  `json:"id"`
	Count    int     `json:"count"`
	LastUsed int64   `json:"lastUsed"`
	Score    float64 `json:"score"`
}

// FrecencyStore persists and computes frecency scores (frequency × recency).
type FrecencyStore struct {
	mu      sync.Mutex
	records map[string]*frecencyRecord
	path    string
}

// NewFrecencyStore creates a frecency store backed by a JSON file.
func NewFrecencyStore(dataDir string) *FrecencyStore {
	s := &FrecencyStore{
		records: make(map[string]*frecencyRecord),
		path:    filepath.Join(dataDir, "frecency.json"),
	}
	s.load()
	return s
}

// RecordAccess records an access for the given ID.
func (s *FrecencyStore) RecordAccess(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.records[id]
	if !ok {
		r = &frecencyRecord{ID: id}
		s.records[id] = r
	}
	r.Count++
	r.LastUsed = time.Now().UnixMilli()
	r.Score = s.computeScore(r)
	s.save()
}

// GetScore returns the frecency score for an ID (higher = more recent & frequent).
func (s *FrecencyStore) GetScore(id string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.records[id]; ok {
		return r.Score
	}
	return 0
}

func (s *FrecencyStore) computeScore(r *frecencyRecord) float64 {
	ageMs := float64(time.Now().UnixMilli() - r.LastUsed)
	if ageMs < 0 {
		ageMs = 0
	}
	ageHours := ageMs / (1000 * 3600)
	return float64(r.Count) / (1 + ageHours*0.1)
}

func (s *FrecencyStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var records map[string]*frecencyRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}
	s.records = records
}

func (s *FrecencyStore) save() {
	data, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0644)
}

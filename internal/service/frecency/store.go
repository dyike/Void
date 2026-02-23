package frecency

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Record captures usage data for frecency scoring.
type Record struct {
	ID       string  `json:"id"`
	Count    int     `json:"count"`
	LastUsed int64   `json:"lastUsed"`
	Score    float64 `json:"score"`
}

// Store persists and computes frecency scores (frequency × recency).
type Store struct {
	mu      sync.Mutex
	records map[string]*Record
	path    string
}

// NewStore creates a frecency store backed by a JSON file.
func NewStore(dataDir string) *Store {
	s := &Store{
		records: make(map[string]*Record),
		path:    filepath.Join(dataDir, "frecency.json"),
	}
	s.load()
	return s
}

// Record an access for the given ID.
func (s *Store) RecordAccess(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.records[id]
	if !ok {
		r = &Record{ID: id}
		s.records[id] = r
	}
	r.Count++
	r.LastUsed = time.Now().UnixMilli()
	r.Score = s.computeScore(r)
	s.save()
}

// GetScore returns the frecency score for an ID (higher = more recent & frequent).
func (s *Store) GetScore(id string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.records[id]; ok {
		return r.Score
	}
	return 0
}

// TopIDs returns the top N most recently/frequently used IDs.
func (s *Store) TopIDs(n int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	type scored struct {
		id    string
		score float64
	}
	var items []scored
	for id, r := range s.records {
		items = append(items, scored{id, r.Score})
	}
	// sort desc
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].score > items[i].score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	result := make([]string, 0, n)
	for i, item := range items {
		if i >= n {
			break
		}
		result = append(result, item.id)
	}
	return result
}

// computeScore uses a simple half-life decay: score = count / (1 + age_hours)^0.5
func (s *Store) computeScore(r *Record) float64 {
	ageMs := float64(time.Now().UnixMilli() - r.LastUsed)
	if ageMs < 0 {
		ageMs = 0
	}
	ageHours := ageMs / (1000 * 3600)
	return float64(r.Count) / (1 + ageHours*0.1)
}

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var records map[string]*Record
	if err := json.Unmarshal(data, &records); err != nil {
		return
	}
	s.records = records
}

func (s *Store) save() {
	data, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0644)
}

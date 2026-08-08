package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Episode struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Text      string `json:"text,omitempty"`
	Script    string `json:"script,omitempty"`
	MP3Path   string `json:"mp3Path,omitempty"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func New(path string) (*Store, error) {
	s := &Store{path: path}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) List() ([]Episode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list()
}

func (s *Store) Get(id string) (Episode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	episodes, err := s.list()
	if err != nil {
		return Episode{}, false
	}
	for _, e := range episodes {
		if e.ID == id {
			return e, true
		}
	}
	return Episode{}, false
}

func (s *Store) list() ([]Episode, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var episodes []Episode
	if err := json.Unmarshal(raw, &episodes); err != nil {
		return nil, err
	}
	return episodes, nil
}

func (s *Store) save(episodes []Episode) error {
	raw, err := json.MarshalIndent(episodes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

func (s *Store) Upsert(ep Episode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	episodes, err := s.list()
	if err != nil {
		return err
	}
	idx := -1
	for i, e := range episodes {
		if e.ID == ep.ID {
			idx = i
			break
		}
	}
	if idx >= 0 {
		episodes[idx] = ep
	} else {
		episodes = append([]Episode{ep}, episodes...)
	}
	return s.save(episodes)
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

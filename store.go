package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Result struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Answers  string    `json:"answers"`
	Duration int       `json:"duration"`
	Created  time.Time `json:"created"`
}

type Store struct {
	mu   sync.RWMutex
	file string
	data []Result
}

func NewStore(file string) (*Store, error) {
	s := &Store{file: file}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = []Result{}
			return nil
		}
		return err
	}
	if len(b) == 0 {
		s.data = []Result{}
		return nil
	}
	return json.Unmarshal(b, &s.data)
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.file), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.WriteFile(s.file, b, 0o644); err != nil {
		return err
	}
	return nil
}

func (s *Store) Add(result Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	prevLen := len(s.data)
	s.data = append(s.data, result)
	if err := s.persistLocked(); err != nil {
		s.data = s.data[:prevLen]
		return err
	}
	return nil
}

func (s *Store) All() ([]Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Result, len(s.data))
	copy(out, s.data)
	return out, nil
}

func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.data
	s.data = []Result{}
	if err := s.persistLocked(); err != nil {
		s.data = prev
		return err
	}
	return nil
}

func (s *Store) DeleteByID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, r := range s.data {
		if r.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return os.ErrNotExist
	}

	next := make([]Result, 0, len(s.data)-1)
	next = append(next, s.data[:idx]...)
	next = append(next, s.data[idx+1:]...)

	prev := s.data
	s.data = next
	if err := s.persistLocked(); err != nil {
		s.data = prev
		return err
	}
	return nil
}

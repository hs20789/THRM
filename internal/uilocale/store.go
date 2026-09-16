package uilocale

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store is owned by Core, independent of AppConfig and GUI lifetime.
type Store struct {
	mu     sync.RWMutex
	locale string
	path   string
}

func NewStore(configDir string) *Store {
	s := &Store{locale: Default}
	if configDir == "" {
		return s
	}
	s.path = filepath.Join(configDir, "ui-locale.json")
	var saved struct {
		Locale string `json:"locale"`
	}
	if data, err := os.ReadFile(s.path); err == nil && json.Unmarshal(data, &saved) == nil {
		s.locale = Normalize(saved.Locale)
	}
	return s
}

func (s *Store) Snapshot() Snapshot {
	if s == nil {
		return Snapshot(Default)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot(Normalize(s.locale))
}

func (s *Store) Set(locale string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	locale = Normalize(locale)
	if locale == s.locale {
		return nil
	}
	if s.path == "" {
		return fmt.Errorf("UI locale storage directory is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(struct {
		Locale string `json:"locale"`
	}{locale})
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".ui-locale-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	s.locale = locale
	return nil
}

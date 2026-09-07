package main

import (
	"errors"
	"sort"
	"sync"
)

// Store is the single thread-safe in-memory data source for feature flags.
// It holds exclusively flag data (key, enabled, description, rollout_percent);
// user identifiers from evaluation requests are never stored here.
type Store struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

// store is the package-level store shared by every handler.
var store = NewStore()

var (
	// ErrKeyExists is returned by Create when the key is already present.
	ErrKeyExists = errors.New("key already exists")
	// ErrKeyNotFound is returned by Update when the key is unknown.
	ErrKeyNotFound = errors.New("key not found")
)

// NewStore returns an empty, ready-to-use Store.
func NewStore() *Store {
	return &Store{flags: make(map[string]Flag)}
}

// Create adds a flag. It fails with ErrKeyExists if the key already exists.
func (s *Store) Create(f Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.flags[f.Key]; exists {
		return ErrKeyExists
	}
	s.flags[f.Key] = f
	return nil
}

// List returns all flags sorted by key.
func (s *Store) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flags := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	sort.Slice(flags, func(i, j int) bool { return flags[i].Key < flags[j].Key })
	return flags
}

// Get returns the flag for key and whether it exists.
func (s *Store) Get(key string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flags[key]
	return f, ok
}

// Update applies upd to the flag for key. Missing optional fields (nil
// Description / RolloutPercent) leave the existing values unchanged. It fails
// with ErrKeyNotFound for an unknown key.
func (s *Store) Update(key string, upd flagUpdate) (Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.flags[key]
	if !ok {
		return Flag{}, ErrKeyNotFound
	}
	f.Enabled = upd.Enabled
	if upd.Description != nil {
		f.Description = *upd.Description
	}
	if upd.RolloutPercent != nil {
		f.RolloutPercent = *upd.RolloutPercent
	}
	s.flags[key] = f
	return f, nil
}

// Delete removes the flag for key and reports whether it was present.
func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[key]; !ok {
		return false
	}
	delete(s.flags, key)
	return true
}

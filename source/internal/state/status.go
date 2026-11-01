package state

import (
	"errors"
	"sync"
)

// Status state and its locking are provided before Week 3 registry work.
type Status struct {
	mu    sync.RWMutex
	value string
}

func NewStatus() *Status      { return &Status{value: "healthy"} }
func (s *Status) Get() string { s.mu.RLock(); defer s.mu.RUnlock(); return s.value }
func (s *Status) Set(v string) error {
	if v != "healthy" && v != "degraded" {
		return errors.New("status must be healthy or degraded")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = v
	return nil
}

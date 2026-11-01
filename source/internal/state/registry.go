package state

import (
	"errors"
	"sort"
	"sync"

	"example.edu/6cc545/cw1/internal/model"
)

var ErrDuplicate = errors.New("duplicate peer ID")
var ErrMissing = errors.New("peer not found")

type Registry struct {
	mu    sync.RWMutex
	peers map[string]model.Peer
}

func NewRegistry() *Registry { return &Registry{peers: make(map[string]model.Peer)} }

func (s *Registry) Add(p model.Peer) error {
	// Validate the peer before modifying the registry.
	if err := p.Validate(); err != nil {
		return err
	}

	// Lock the registry while checking and inserting to prevent
	// concurrent writers from adding the same peer ID.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reject duplicate peer IDs without overwriting existing records.
	if _, exists := s.peers[p.ID]; exists {
		return ErrDuplicate
	}

	// Store the validated peer in the registry.
	s.peers[p.ID] = p
	return nil
}

func (s *Registry) Snapshot() []model.Peer {
	// Acquire a read lock while copying the registry's records.
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a new slice so callers cannot modify the registry
	// by changing the returned snapshot.
	peers := make([]model.Peer, 0, len(s.peers))
	for _, p := range s.peers {
		peers = append(peers, p)
	}

	// Sort the copied records by peer ID for deterministic output.
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].ID < peers[j].ID
	})

	return peers
}

func (s *Registry) Get(id string) (model.Peer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.peers[id]
	if !ok {
		return p, ErrMissing
	}
	return p, nil
}

var _ = sort.Slice

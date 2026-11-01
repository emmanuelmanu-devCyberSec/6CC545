package state

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"example.edu/6cc545/cw1/internal/model"
)

func peer(id string) model.Peer {
	return model.Peer{ID: id, HTTPAddress: "127.0.0.1:8082", GRPCAddress: "127.0.0.1:9092", Status: "unknown"}
}
func TestWeek03_Registry(t *testing.T) {
	s := NewRegistry()
	for _, id := range []string{"z", "a"} {
		if e := s.Add(peer(id)); e != nil {
			t.Fatal(e)
		}
	}
	if !errors.Is(s.Add(peer("a")), ErrDuplicate) {
		t.Fatal("duplicate policy must be explicit")
	}
	p := s.Snapshot()
	if len(p) != 2 || p[0].ID != "a" {
		t.Fatal("snapshot must sort by ID")
	}
	p[0].ID = "changed"
	if s.Snapshot()[0].ID != "a" {
		t.Fatal("snapshot aliases live state")
	}
}
func TestWeek03_ConcurrentRegistry(t *testing.T) {
	s := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if e := s.Add(peer(fmt.Sprintf("n%02d", i))); e != nil {
				t.Error(e)
			}
			_ = s.Snapshot()
		}(i)
	}
	wg.Wait()
	if n := len(s.Snapshot()); n != 50 {
		t.Fatalf("want 50 peers, got %d", n)
	}
}
func TestWeek03_InvalidPeer(t *testing.T) {
	s := NewRegistry()
	p := peer("x")
	p.HTTPAddress = "broken"
	if s.Add(p) == nil {
		t.Fatal("invalid address accepted")
	}
	if len(s.Snapshot()) != 0 {
		t.Fatal("invalid input changed state")
	}
}

func TestWeek03_ConcurrentDuplicatePeer(t *testing.T) {
	// Use multiple goroutines to attempt to register the same peer ID.
	const workers = 20

	s := NewRegistry()

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Track successful registrations, duplicate errors and unexpected errors.
	successes := 0
	duplicates := 0
	unexpected := 0

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Every goroutine attempts to add the same peer.
			err := s.Add(peer("same-peer"))

			// Protect the counters because goroutines update them concurrently.
			mu.Lock()
			defer mu.Unlock()

			// Exactly one registration should succeed.
			// Every other attempt should return ErrDuplicate.
			switch {
			case err == nil:
				successes++
			case errors.Is(err, ErrDuplicate):
				duplicates++
			default:
				unexpected++
			}
		}()
	}

	// Wait for all registration attempts to finish before checking results.
	wg.Wait()

	// Verify that exactly one goroutine registered the peer successfully.
	if successes != 1 {
		t.Errorf("successes = %d, want 1", successes)
	}

	// Verify that all remaining attempts returned duplicate errors.
	if duplicates != workers-1 {
		t.Errorf("duplicates = %d, want %d", duplicates, workers-1)
	}

	// No other errors should have occurred.
	if unexpected != 0 {
		t.Errorf("unexpected errors = %d, want 0", unexpected)
	}

	// Confirm that the registry contains exactly one peer.
	if got := len(s.Snapshot()); got != 1 {
		t.Errorf("registry count = %d, want 1", got)
	}
}

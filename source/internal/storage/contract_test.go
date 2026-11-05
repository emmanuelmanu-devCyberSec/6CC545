//go:build acceptance

package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
)

func digest(b []byte) string { x := sha256.Sum256(b); return hex.EncodeToString(x[:]) }
func TestPortableNames(t *testing.T) {
	for _, n := range []string{"notes.txt", "A_1-b.csv"} {
		if e := ValidateName(n); e != nil {
			t.Errorf("valid %q: %v", n, e)
		}
	}
	for _, n := range []string{"", "../x", "a/b", "a\\b", ".hidden", "a..b", "CON", "nul.txt", "COM1.csv", "x.", "x:y"} {
		if e := ValidateName(n); e == nil {
			t.Errorf("unsafe %q accepted", n)
		}
	}
}
func TestCommitRoundTripAndIsolation(t *testing.T) {
	s, _ := New(t.TempDir(), 1024)
	body := []byte("hello\n")
	m, e := s.Put(context.Background(), "a.txt", bytes.NewReader(body), int64(len(body)), digest(body))
	if e != nil {
		t.Fatal(e)
	}
	r, got, e := s.Open("a.txt")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	b, e := io.ReadAll(r)
	if e != nil || !bytes.Equal(b, body) || got != m {
		t.Fatalf("round trip: %q %+v %v", b, got, e)
	}
	list, e := s.Snapshot()
	if e != nil || len(list) != 1 {
		t.Fatalf("snapshot %v %v", list, e)
	}
	list[0].Name = "changed"
	list2, e := s.Snapshot()
	if e != nil || list2[0].Name != "a.txt" {
		t.Fatal("snapshot aliases state")
	}
}
func TestRejectedWritePreservesCommittedFile(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir, 16)
	ctx := context.Background()
	old := []byte("old")
	if _, e := s.Put(ctx, "safe.txt", bytes.NewReader(old), 3, digest(old)); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		b []byte
		n int64
		h string
	}{{[]byte("bad"), 3, digest(old)}, {[]byte("extra"), 2, digest([]byte("ex"))}, {bytes.Repeat([]byte("x"), 17), 17, digest(bytes.Repeat([]byte("x"), 17))}} {
		if _, e := s.Put(ctx, "safe.txt", bytes.NewReader(tc.b), tc.n, tc.h); e == nil {
			t.Fatal("invalid write accepted")
		}
		r, _, e := s.Open("safe.txt")
		if e != nil {
			t.Fatal(e)
		}
		b, _ := io.ReadAll(r)
		r.Close()
		if !bytes.Equal(b, old) {
			t.Fatal("failed write changed data")
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temporary files leaked: %v", entries)
	}
}
func TestCancellationAndAbsentFile(t *testing.T) {
	s, _ := New(t.TempDir(), 1024)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.Put(ctx, "x.txt", bytes.NewReader([]byte("x")), 1, digest([]byte("x"))); !errors.Is(e, context.Canceled) {
		t.Fatalf("want cancellation, got %v", e)
	}
	if _, _, e := s.Open("missing.txt"); !errors.Is(e, os.ErrNotExist) {
		t.Fatalf("want not-exist, got %v", e)
	}
}
func TestConcurrentSameNameConsistentSnapshot(t *testing.T) {
	s, _ := New(t.TempDir(), 1024)
	seed := []byte("seed")
	if _, e := s.Put(context.Background(), "same.txt", bytes.NewReader(seed), 4, digest(seed)); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := []byte(fmt.Sprintf("body-%02d", i))
			if _, e := s.Put(context.Background(), "same.txt", bytes.NewReader(b), int64(len(b)), digest(b)); e != nil {
				t.Error(e)
			}
			r, m, e := s.Open("same.txt")
			if e != nil {
				t.Error(e)
				return
			}
			got, e := io.ReadAll(r)
			r.Close()
			if e != nil || int64(len(got)) != m.Size || digest(got) != m.SHA256 {
				t.Error("inconsistent file snapshot")
			}
		}(i)
	}
	wg.Wait()
}

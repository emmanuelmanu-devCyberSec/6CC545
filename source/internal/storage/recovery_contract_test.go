//go:build acceptance

package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverCommittedData(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir, 1024)
	b := []byte("persist")
	if _, e := s.Put(context.Background(), "keep.txt", bytes.NewReader(b), int64(len(b)), digest(b)); e != nil {
		t.Fatal(e)
	}
	s2, _ := New(dir, 1024)
	if e := s2.Recover(context.Background()); e != nil {
		t.Fatal(e)
	}
	r, m, e := s2.Open("keep.txt")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, b) || m.SHA256 != digest(b) {
		t.Fatal("recovery changed committed file")
	}
	if _, e = os.Stat(filepath.Join(dir, "keep.txt")); e != nil {
		t.Fatal(e)
	}
}

package storage

import "testing"

func TestScaffoldDirectory(t *testing.T) {
	s, e := New(t.TempDir(), 1024)
	if e != nil || s == nil {
		t.Fatal(e)
	}
	if _, e = New("", 1024); e == nil {
		t.Fatal("empty root accepted")
	}
}

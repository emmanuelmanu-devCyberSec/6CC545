package ipc

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
)

type fragmentReader struct{ r io.Reader }

func (f fragmentReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return f.r.Read(p)
}

type shortWriter struct{ b bytes.Buffer }

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) > 2 {
		p = p[:2]
	}
	return w.b.Write(p)
}
func TestWeek04_FrameFragmentation(t *testing.T) {
	var w shortWriter
	want := []byte(`{"text":"fragmented"}`)
	if e := WriteFrame(&w, want); e != nil {
		t.Fatal(e)
	}
	got, e := ReadFrame(fragmentReader{&w.b})
	if e != nil || !bytes.Equal(got, want) {
		t.Fatalf("frame lost bytes: %q %v", got, e)
	}
}
func TestWeek04_FrameLimits(t *testing.T) {
	for _, n := range []uint32{0, MaxMessage + 1} {
		var h [4]byte
		binary.BigEndian.PutUint32(h[:], n)
		if _, e := ReadFrame(bytes.NewReader(h[:])); e == nil {
			t.Fatalf("length %d accepted", n)
		}
	}
	if e := WriteFrame(io.Discard, bytes.Repeat([]byte{'x'}, MaxMessage+1)); e == nil {
		t.Fatal("oversized write accepted")
	}
}
func TestWeek04_InvalidMessage(t *testing.T) {
	valid := Message{1, "r1", "node-a", "hello"}
	if e := valid.Validate(); e != nil {
		t.Fatal(e)
	}
	valid.Version = 2
	if valid.Validate() == nil {
		t.Fatal("version 2 accepted")
	}
	for _, s := range []string{`{bad}`, `{"surprise":1}`, `{}{}`, strings.Repeat("x", MaxMessage+1)} {
		var m Message
		if Decode([]byte(s), &m) == nil {
			t.Fatalf("invalid data accepted: %.50s", s)
		}
	}
}

func TestWeek04_MessageValidationBoundaries(t *testing.T) {
	// Start with a valid message for the boundary checks.
	valid := Message{
		Version:   Version,
		RequestID: "r1",
		NodeID:    "node-a",
		Text:      "hello",
	}

	// Check that empty text is rejected.
	emptyText := valid
	emptyText.Text = ""
	if err := emptyText.Validate(); err == nil {
		t.Fatal("empty text accepted")
	}

	// Check that an invalid request ID is rejected.
	invalidID := valid
	invalidID.RequestID = ""
	if err := invalidID.Validate(); err == nil {
		t.Fatal("empty request ID accepted")
	}

	// Check that text exceeding 512 bytes is rejected.
	oversizedText := valid
	oversizedText.Text = strings.Repeat("a", 513)
	if err := oversizedText.Validate(); err == nil {
		t.Fatal("text exceeding 512 bytes accepted")
	}

	// Confirm that text at the 512-byte limit is accepted.
	maxText := valid
	maxText.Text = strings.Repeat("a", 512)
	if err := maxText.Validate(); err != nil {
		t.Fatalf("512-byte text rejected: %v", err)
	}
}

func TestWeek04_IncompleteFrameHeader(t *testing.T) {
	// Supply fewer than four bytes for the frame header.
	_, err := ReadFrame(bytes.NewReader([]byte{0, 0}))
	if err == nil {
		t.Fatal("incomplete frame header accepted")
	}
}

func TestWeek04_IncompleteFrameBody(t *testing.T) {
	// Declare a five-byte payload but provide only two bytes.
	data := []byte{0, 0, 0, 5, 'h', 'i'}

	_, err := ReadFrame(bytes.NewReader(data))
	if err == nil {
		t.Fatal("incomplete frame body accepted")
	}
}

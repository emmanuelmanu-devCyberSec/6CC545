package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/events"
	"example.edu/6cc545/cw1/internal/storage"
)

const testUploadLimit int64 = 1 << 20

func newFileUploadRequest(
	t *testing.T,
	name string,
	content []byte,
	digest string,
) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filePart, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := filePart.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := writer.WriteField("sha256", digest); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/files",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func newFileUploadRoute(
	t *testing.T,
	bus *events.Bus,
) (*http.ServeMux, *storage.Store, string) {
	t.Helper()

	root := t.TempDir()

	store, err := storage.New(root, testUploadLimit)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()

	_, _, err = base(
		config.Config{ID: "test-node"},
		mux,
		bus,
		store,
		testUploadLimit,
	)
	if err != nil {
		t.Fatal(err)
	}

	return mux, store, root
}

func TestWeek07_FileAddedOnlyAfterVerifiedUpload(t *testing.T) {
	bus, err := events.New(16)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	sub, err := bus.Subscribe("file.added")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Cancel()

	mux, store, root := newFileUploadRoute(t, bus)

	content := []byte("verified coursework upload")
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])

	// A valid upload should be committed and published.
	req := newFileUploadRequest(
		t,
		"valid.txt",
		content,
		digest,
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"valid upload returned %d: %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	select {
	case event := <-sub.C:
		if event.Kind != "file.added" {
			t.Fatalf("unexpected event kind: %q", event.Kind)
		}
		if event.Name != "valid.txt" {
			t.Fatalf("unexpected event name: %q", event.Name)
		}
		if event.Origin != "test-node" {
			t.Fatalf("unexpected event origin: %q", event.Origin)
		}
	case <-time.After(time.Second):
		t.Fatal("no file.added event received")
	default:
		t.Fatal("valid upload did not publish an event")
	}

	// An incorrect digest must be rejected without publishing an event.
	invalidReq := newFileUploadRequest(
		t,
		"invalid.txt",
		[]byte("content with an incorrect digest"),
		strings.Repeat("0", 64),
	)
	invalidRec := httptest.NewRecorder()
	mux.ServeHTTP(invalidRec, invalidReq)

	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf(
			"invalid upload returned %d: %s",
			invalidRec.Code,
			invalidRec.Body.String(),
		)
	}

	if _, err := os.Stat(filepath.Join(root, "invalid.txt")); !os.IsNotExist(err) {
		t.Fatalf("rejected file should not exist, stat error: %v", err)
	}

	select {
	case event := <-sub.C:
		t.Fatalf("rejected upload unexpectedly published an event: %+v", event)
	default:
		// No event was generated for the rejected upload.
	}

	// Confirm the earlier valid file is still stored.
	if _, err := os.Stat(filepath.Join(root, "valid.txt")); err != nil {
		t.Fatalf("committed file is missing: %v", err)
	}

	opened, _, err := store.Open("valid.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
}

func TestWeek07_DroppedNotificationDoesNotUndoCommit(t *testing.T) {
	bus, err := events.New(16)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	sub, err := bus.Subscribe("file.added")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Cancel()

	// Fill the subscriber queue so the next notification is dropped.
	for i := 0; i < 16; i++ {
		_, err := bus.Publish(events.Event{
			ID:     fmt.Sprintf("filler-%d", i),
			Kind:   "file.added",
			Origin: "test-node",
			Name:   fmt.Sprintf("filler-%d.txt", i),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	mux, _, root := newFileUploadRoute(t, bus)

	content := []byte("committed even when notification is dropped")
	sum := sha256.Sum256(content)

	req := newFileUploadRequest(
		t,
		"dropped-event.txt",
		content,
		hex.EncodeToString(sum[:]),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"upload returned %d: %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	var response struct {
		Dropped        int  `json:"dropped"`
		Delivered      int  `json:"delivered"`
		EventPublished bool `json:"event_published"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.Dropped != 1 {
		t.Fatalf("expected one dropped notification, got %d", response.Dropped)
	}

	if !response.EventPublished {
		t.Fatal("publish call should succeed even when a subscriber queue drops the event")
	}

	// A notification failure must not remove a committed file.
	if _, err := os.Stat(filepath.Join(root, "dropped-event.txt")); err != nil {
		t.Fatalf("committed file was removed after notification drop: %v", err)
	}
}

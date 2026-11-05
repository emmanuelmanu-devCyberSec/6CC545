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
	"time"
)

func concurrencyDigest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func TestConcurrentIndependentFiles(t *testing.T) {
	s, err := New(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			<-start

			name := fmt.Sprintf("file-%02d.txt", i)
			body := []byte(fmt.Sprintf("content-%02d", i))

			_, err := s.Put(
				context.Background(),
				name,
				bytes.NewReader(body),
				int64(len(body)),
				concurrencyDigest(body),
			)
			if err != nil {
				errs <- err
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	for i := 0; i < workers; i++ {
		name := fmt.Sprintf("file-%02d.txt", i)
		expected := []byte(fmt.Sprintf("content-%02d", i))

		r, metadata, err := s.Open(name)
		if err != nil {
			t.Fatal(err)
		}

		actual, err := io.ReadAll(r)
		closeErr := r.Close()

		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("incorrect contents for %s", name)
		}
		if metadata.Size != int64(len(expected)) ||
			metadata.SHA256 != concurrencyDigest(expected) {
			t.Errorf("incorrect metadata for %s", name)
		}
	}
}

func TestConcurrentSameNameCompleteVersions(t *testing.T) {
	s, err := New(t.TempDir(), 4096)
	if err != nil {
		t.Fatal(err)
	}

	seed := []byte("initial-version")
	if _, err := s.Put(
		context.Background(), "shared.txt",
		bytes.NewReader(seed), int64(len(seed)), concurrencyDigest(seed),
	); err != nil {
		t.Fatal(err)
	}

	// Every reader must receive one of these complete versions.
	knownVersions := map[string]bool{
		string(seed): true,
	}

	const workers = 8
	const readsPerWorker = 20

	for i := 0; i < workers; i++ {
		body := []byte(fmt.Sprintf("writer-%02d-complete-version", i))
		knownVersions[string(body)] = true
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, workers*(readsPerWorker+1))

	// Writers repeatedly replace the same filename.
	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			<-start

			body := []byte(fmt.Sprintf("writer-%02d-complete-version", i))

			_, err := s.Put(
				context.Background(),
				"shared.txt",
				bytes.NewReader(body),
				int64(len(body)),
				concurrencyDigest(body),
			)
			if err != nil {
				errs <- err
			}
		}(i)
	}

	// Readers verify the actual contents and matching metadata.
	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < readsPerWorker; j++ {
				r, metadata, err := s.Open("shared.txt")
				if err != nil {
					errs <- err
					return
				}

				body, readErr := io.ReadAll(r)
				closeErr := r.Close()

				if readErr != nil {
					errs <- readErr
					return
				}
				if closeErr != nil {
					errs <- closeErr
					return
				}

				if !knownVersions[string(body)] {
					errs <- errors.New("reader received an incomplete or unexpected version")
					return
				}

				if metadata.Size != int64(len(body)) ||
					metadata.SHA256 != concurrencyDigest(body) {
					errs <- errors.New("file contents do not match returned metadata")
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

type cancelDuringRead struct {
	cancel context.CancelFunc
}

func (r cancelDuringRead) Read([]byte) (int, error) {
	r.cancel()
	return 0, context.Canceled
}

func TestCancelledWriteDoesNotHoldStoreLock(t *testing.T) {
	s, err := New(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}

	original := []byte("original-version")
	if _, err := s.Put(
		context.Background(), "shared.txt",
		bytes.NewReader(original), int64(len(original)),
		concurrencyDigest(original),
	); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	_, err = s.Put(
		ctx,
		"shared.txt",
		cancelDuringRead{cancel: cancel},
		1,
		concurrencyDigest([]byte("x")),
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	// The previous committed version must remain available.
	r, metadata, err := s.Open("shared.txt")
	if err != nil {
		t.Fatal(err)
	}

	actual, err := io.ReadAll(r)
	r.Close()

	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, original) ||
		metadata.SHA256 != concurrencyDigest(original) {
		t.Fatal("cancelled write changed the previous file")
	}

	// There should be no abandoned temporary file.
	entries, err := os.ReadDir(s.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected files after cancellation: %v", entries)
	}

	// A subsequent write must be able to acquire the store lock.
	done := make(chan error, 1)

	go func() {
		body := []byte("next-write")
		_, err := s.Put(
			context.Background(),
			"next.txt",
			bytes.NewReader(body),
			int64(len(body)),
			concurrencyDigest(body),
		)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("store lock appears to remain held after cancellation")
	}
}

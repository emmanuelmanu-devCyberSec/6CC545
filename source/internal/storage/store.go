// Package storage is an assessed single-process local file store wireframe.
package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var ErrNotImplemented = errors.New("TODO W07: storage not implemented")
var ErrInvalidName = errors.New("invalid portable file name")
var ErrIntegrity = errors.New("size or SHA-256 mismatch")
var ErrTooLarge = errors.New("file exceeds configured limit")

type Metadata struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type Store struct {
	mu       sync.RWMutex
	root     string
	maxBytes int64
	files    map[string]Metadata
}

// New is supplied directory allocation, not a persistence/recovery implementation.
// The directory must be owned exclusively by this process and not attacker-writable.
// Task 8.6 adds recovery of committed files after a restart.
func New(root string, maxBytes int64) (*Store, error) {
	if root == "" || maxBytes < 1 {
		return nil, errors.New("root and positive maxBytes required")
	}
	abs, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(abs, 0700); e != nil {
		return nil, e
	}
	info, e := os.Lstat(abs)
	if e != nil {
		return nil, e
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("root must be a directory, not a symlink")
	}
	return &Store{root: abs, maxBytes: maxBytes, files: make(map[string]Metadata)}, nil
}

func ValidateName(name string) error {
	// Keep filenames within the portable naming rules defined in the contract.
	// This also prevents unusually long names from reaching the filesystem.
	if len(name) == 0 || len(name) > 128 {
		return ErrInvalidName
	}

	// Filenames must begin with a letter or number.
	// This prevents hidden names and names beginning with punctuation.
	first := name[0]
	if !isLetterOrDigit(first) {
		return ErrInvalidName
	}

	// Reject traversal patterns and trailing dots instead of trying to
	// clean up the supplied name. Different inputs must not become the same file.
	if name == "." || name == ".." ||
		strings.Contains(name, "..") ||
		strings.HasSuffix(name, ".") {
		return ErrInvalidName
	}

	// Only accept characters that are safe and consistent across platforms.
	for i := 0; i < len(name); i++ {
		c := name[i]

		if !isLetterOrDigit(c) && c != '.' && c != '_' && c != '-' {
			return ErrInvalidName
		}
	}

	// Windows reserves certain device names. These remain reserved even
	// when a file extension is added, so check the part before the first dot.
	base := name
	if dot := strings.IndexByte(name, '.'); dot >= 0 {
		base = name[:dot]
	}

	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5",
		"COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5",
		"LPT6", "LPT7", "LPT8", "LPT9":
		return ErrInvalidName
	}

	return nil
}

// This helper keeps the character checks in one place and makes the
// filename validation easier to read.
func isLetterOrDigit(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}
func (s *Store) Put(ctx context.Context, name string, r io.Reader, size int64, sha string) (Metadata, error) {
	// Check the supplied details before creating a temporary file.
	if err := ValidateName(name); err != nil {
		return Metadata{}, err
	}

	if ctx == nil || r == nil {
		return Metadata{}, errors.New("context and reader are required")
	}

	if size < 0 {
		return Metadata{}, ErrIntegrity
	}

	if size > s.maxBytes {
		return Metadata{}, ErrTooLarge
	}

	// A SHA-256 digest must be exactly 64 hexadecimal characters.
	if len(sha) != 64 {
		return Metadata{}, ErrIntegrity
	}

	if _, err := hex.DecodeString(sha); err != nil {
		return Metadata{}, ErrIntegrity
	}

	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}

	// Keep the temporary file in the storage directory so the final rename
	// takes place on the same filesystem.
	temp, err := os.CreateTemp(s.root, ".w07-tmp-*")
	if err != nil {
		return Metadata{}, err
	}

	tempName := temp.Name()
	committed := false
	closed := false

	// Remove the temporary file whenever the write does not commit.
	defer func() {
		if !closed {
			_ = temp.Close()
		}
		if !committed {
			_ = os.Remove(tempName)
		}
	}()

	hasher := sha256.New()
	reader := &contextReader{ctx: ctx, reader: r}

	// Copy no more than the expected size while calculating the digest.
	written, err := io.Copy(
		io.MultiWriter(temp, hasher),
		io.LimitReader(reader, size),
	)
	if err != nil {
		return Metadata{}, err
	}

	if written != size {
		return Metadata{}, ErrIntegrity
	}

	// Check for an extra byte so oversized bodies are rejected.
	var extra [1]byte
	n, err := io.ReadFull(reader, extra[:])
	if n > 0 {
		return Metadata{}, ErrIntegrity
	}
	if err != io.EOF {
		return Metadata{}, err
	}

	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != strings.ToLower(sha) {
		return Metadata{}, ErrIntegrity
	}

	// Flush and close the temporary file before making it visible.
	if err := temp.Sync(); err != nil {
		return Metadata{}, err
	}

	if err := temp.Close(); err != nil {
		closed = true
		return Metadata{}, err
	}
	closed = true

	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}

	metadata := Metadata{
		Name:   name,
		Size:   written,
		SHA256: actualHash,
	}

	// Protect the final replacement and metadata update as one commit.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}

	if err := os.Rename(tempName, filepath.Join(s.root, name)); err != nil {
		return Metadata{}, err
	}

	committed = true
	s.files[name] = metadata

	return metadata, nil
}

// Check cancellation before reading more data.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
func (s *Store) Open(name string) (io.ReadCloser, Metadata, error) {
	if err := ValidateName(name); err != nil {
		return nil, Metadata{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, exists := s.files[name]
	if !exists {
		return nil, Metadata{}, os.ErrNotExist
	}

	file, err := os.Open(filepath.Join(s.root, name))
	if err != nil {
		return nil, Metadata{}, err
	}

	// Read the committed version while holding the lock so the
	// contents and metadata belong to the same version.
	data, err := io.ReadAll(io.LimitReader(file, metadata.Size+1))
	closeErr := file.Close()

	if err != nil {
		return nil, Metadata{}, err
	}
	if closeErr != nil {
		return nil, Metadata{}, closeErr
	}

	if int64(len(data)) != metadata.Size {
		return nil, Metadata{}, ErrIntegrity
	}

	actualHash := sha256.Sum256(data)
	if hex.EncodeToString(actualHash[:]) != metadata.SHA256 {
		return nil, Metadata{}, ErrIntegrity
	}

	return io.NopCloser(strings.NewReader(string(data))), metadata, nil
}

func (s *Store) Snapshot() ([]Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Copy the catalogue so callers cannot modify the store's internal data.
	snapshot := make([]Metadata, 0, len(s.files))

	for _, metadata := range s.files {
		snapshot = append(snapshot, metadata)
	}

	// Keep the result in a predictable order.
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].Name < snapshot[j].Name
	})

	return snapshot, nil
}
func (s *Store) Recover(ctx context.Context) error {
	// TODO 8.6: rebuild committed metadata on restart, ignore/remove abandoned
	// temp files, revalidate names/sizes/hashes; do not follow symlinks. A clear
	// private-temp naming convention is needed. Never infer validity from name.
	return ErrNotImplemented
}

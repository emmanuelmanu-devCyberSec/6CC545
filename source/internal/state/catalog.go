package state

import (
	"errors"
	"sort"
	"sync"

	"example.edu/6cc545/cw1/internal/model"
)

// Catalog contains metadata, not mutable file contents. File I/O belongs in CW2.
type Catalog struct {
	mu    sync.RWMutex
	files []model.FileMetadata
}

func NewCatalog() *Catalog { return &Catalog{files: []model.FileMetadata{}} }

// ErrDuplicateFile is returned when a filename already exists in the catalogue.
var ErrDuplicateFile = errors.New("duplicate filename")

// Add validates and inserts one file metadata record into the catalogue.
func (c *Catalog) Add(f model.FileMetadata) error {
	// Validate the metadata before modifying the catalogue.
	if err := f.Validate(); err != nil {
		return err
	}

	// Lock the catalogue to prevent concurrent updates.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reject duplicate filenames without replacing existing records.
	for _, existing := range c.files {
		if existing.Name == f.Name {
			return ErrDuplicateFile
		}
	}

	// Add the validated metadata record.
	c.files = append(c.files, f)

	// Keep the catalogue sorted by filename.
	sort.Slice(c.files, func(i, j int) bool {
		return c.files[i].Name < c.files[j].Name
	})

	return nil
}

func (c *Catalog) Replace(input []model.FileMetadata) error {
	// Copy the input so changes to the caller's slice cannot affect
	// the catalogue while validation is in progress.
	candidate := append([]model.FileMetadata{}, input...)

	// Validate every record and reject duplicate filenames before
	// modifying the existing catalogue.
	seen := make(map[string]bool, len(candidate))
	for _, f := range candidate {
		if err := f.Validate(); err != nil {
			return err
		}

		if seen[f.Name] {
			return errors.New("duplicate filename")
		}
		seen[f.Name] = true
	}

	// Sort the validated candidate by filename for deterministic output.
	sort.Slice(candidate, func(i, j int) bool {
		return candidate[i].Name < candidate[j].Name
	})

	// Replace the current catalogue atomically under the write lock.
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = candidate

	return nil
}

func (c *Catalog) Snapshot() []model.FileMetadata {
	// Acquire a read lock while copying the current metadata.
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return an independent slice so callers cannot modify the
	// catalogue by changing the returned snapshot.
	files := make([]model.FileMetadata, len(c.files))
	copy(files, c.files)

	// Keep the snapshot ordered by filename.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files
}

var _ = errors.New
var _ = sort.Slice

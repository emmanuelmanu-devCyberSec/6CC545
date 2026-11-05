package httpapi

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"example.edu/6cc545/cw1/internal/events"
	"example.edu/6cc545/cw1/internal/model"
	"example.edu/6cc545/cw1/internal/state"
	"example.edu/6cc545/cw1/internal/storage"
)

func Peers(s *state.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			WriteJSON(w, http.StatusOK, s.Snapshot())
			return
		}

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		var p model.Peer
		if err := DecodeOne(w, r, &p); err != nil {
			ErrorJSON(w, http.StatusBadRequest, "invalid_request")
			return
		}

		if err := s.Add(p); err != nil {
			if errors.Is(err, state.ErrDuplicate) {
				ErrorJSON(w, http.StatusConflict, "duplicate_peer")
				return
			}

			ErrorJSON(w, http.StatusBadRequest, "invalid_peer")
			return
		}

		WriteJSON(w, http.StatusCreated, p)
	}
}

const multipartOverhead int64 = 1 << 20

func Files(
	c *state.Catalog,
	store *storage.Store,
	maxUploadBytes int64,
	publish func(model.FileMetadata) (string, events.Delivery, error),
) http.HandlerFunc {
	// Keep catalogue updates together when uploads arrive concurrently.
	var uploadMu sync.Mutex

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			WriteJSON(w, http.StatusOK, c.Snapshot())
			return
		}

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		if maxUploadBytes < 1 {
			ErrorJSON(w, http.StatusInternalServerError, "storage_not_configured")
			return
		}

		// Bound the complete multipart request, including its form headers.
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+multipartOverhead)

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				ErrorJSON(w, http.StatusRequestEntityTooLarge, "upload_too_large")
				return
			}

			ErrorJSON(w, http.StatusBadRequest, "invalid_upload")
			return
		}

		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}

		files := r.MultipartForm.File["file"]
		digests := r.MultipartForm.Value["sha256"]

		if len(files) != 1 || len(digests) != 1 {
			ErrorJSON(w, http.StatusBadRequest, "file_and_sha256_required")
			return
		}

		uploaded := files[0]

		if uploaded.Size > maxUploadBytes {
			ErrorJSON(w, http.StatusRequestEntityTooLarge, "upload_too_large")
			return
		}

		file, err := uploaded.Open()
		if err != nil {
			ErrorJSON(w, http.StatusBadRequest, "invalid_upload")
			return
		}
		defer file.Close()

		// Commit the verified file before publishing any event.
		uploadMu.Lock()
		defer uploadMu.Unlock()

		stored, err := store.Put(
			r.Context(),
			uploaded.Filename,
			file,
			uploaded.Size,
			digests[0],
		)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrTooLarge):
				ErrorJSON(w, http.StatusRequestEntityTooLarge, "upload_too_large")
			case errors.Is(err, storage.ErrInvalidName),
				errors.Is(err, storage.ErrIntegrity):
				ErrorJSON(w, http.StatusBadRequest, "file_verification_failed")
			default:
				ErrorJSON(w, http.StatusInternalServerError, "file_storage_failed")
			}
			return
		}

		// Build catalogue metadata from the file that was actually committed.
		metadata := model.FileMetadata{
			Name:         stored.Name,
			Size:         stored.Size,
			SHA256:       stored.SHA256,
			ModifiedUnix: time.Now().Unix(),
		}

		// Replace an existing catalogue entry when the file was updated.
		current := c.Snapshot()
		updated := false

		for i := range current {
			if current[i].Name == metadata.Name {
				current[i] = metadata
				updated = true
				break
			}
		}

		if !updated {
			current = append(current, metadata)
		}

		if err := c.Replace(current); err != nil {
			// The file is already committed. Never remove it because
			// a separate catalogue update failed.
			WriteJSON(w, http.StatusCreated, map[string]any{
				"file":            metadata,
				"catalogue_error": err.Error(),
				"event_published": false,
			})
			return
		}

		// Only now, after a successful commit, publish file.added.
		eventID, delivery, publishErr := publish(metadata)

		response := map[string]any{
			"file":      metadata,
			"event_id":  eventID,
			"delivered": delivery.Delivered,
			"dropped":   delivery.Dropped,
		}

		if publishErr != nil {
			// The file remains committed even if the bus is unavailable.
			response["event_published"] = false
			response["event_error"] = publishErr.Error()
		} else {
			response["event_published"] = true
		}

		WriteJSON(w, http.StatusCreated, response)
	}
}

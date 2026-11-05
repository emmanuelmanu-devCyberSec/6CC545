package httpapi

import (
	"errors"
	"net/http"

	"example.edu/6cc545/cw1/internal/model"
	"example.edu/6cc545/cw1/internal/state"
)

func Peers(s *state.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// GET returns a snapshot of the registered peers.
		if r.Method == http.MethodGet {
			WriteJSON(w, http.StatusOK, s.Snapshot())
			return
		}

		// Only GET and POST are supported.
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		// DecodeOne enforces the existing Week 2 JSON request limits
		// and rejects malformed or oversized request bodies.
		var p model.Peer
		if err := DecodeOne(w, r, &p); err != nil {
			ErrorJSON(w, http.StatusBadRequest, "invalid_request")
			return
		}

		// Add validates the peer and rejects duplicate IDs.
		if err := s.Add(p); err != nil {
			if errors.Is(err, state.ErrDuplicate) {
				ErrorJSON(w, http.StatusConflict, "duplicate_peer")
				return
			}

			ErrorJSON(w, http.StatusBadRequest, "invalid_peer")
			return
		}

		// Return 201 Created when a new peer is registered.
		WriteJSON(w, http.StatusCreated, p)
	}
}

func Files(
	c *state.Catalog,
	publish func(model.FileMetadata) (string, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// GET returns a snapshot of the catalogue.
		if r.Method == http.MethodGet {
			WriteJSON(w, http.StatusOK, c.Snapshot())
			return
		}

		// Only GET and POST are supported.
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		// Decode and validate the incoming metadata.
		var file model.FileMetadata
		if err := DecodeOne(w, r, &file); err != nil {
			ErrorJSON(w, http.StatusBadRequest, "invalid_request")
			return
		}

		// Add the metadata without replacing an existing record.
		if err := c.Add(file); err != nil {
			if errors.Is(err, state.ErrDuplicateFile) {
				ErrorJSON(w, http.StatusConflict, "duplicate_filename")
				return
			}

			ErrorJSON(w, http.StatusBadRequest, "invalid_file_metadata")
			return
		}

		// Publish an event for the newly added metadata.
		eventID, err := publish(file)
		if err != nil {
			ErrorJSON(w, http.StatusServiceUnavailable, "event_publish_failed")
			return
		}

		// Return the created metadata and its event ID.
		WriteJSON(w, http.StatusCreated, map[string]any{
			"file":     file,
			"event_id": eventID,
		})
	}
}

var _ = errors.Is
var _ = model.Peer{}

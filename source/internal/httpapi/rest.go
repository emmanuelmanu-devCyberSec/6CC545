package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"example.edu/6cc545/cw1/internal/state"
)

const MaxBody = 4096

// DecodeOne enforces a single bounded JSON object. No state is mutated here.
func DecodeOne(w http.ResponseWriter, r *http.Request, dst any) error {
	// 1. Classify the media type. Charset parameters are accepted.
	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return &InputError{
			Status: http.StatusUnsupportedMediaType,
			Label:  "unsupported_media_type",
		}
	}

	// 2. Read the body through an actual 4096-byte limit.
	r.Body = http.MaxBytesReader(w, r.Body, MaxBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return &InputError{
				Status: http.StatusRequestEntityTooLarge,
				Label:  "body_too_large",
			}
		}
		return &InputError{
			Status: http.StatusBadRequest,
			Label:  "invalid_json",
		}
	}

	// 3. Require a non-empty JSON object at the top level.
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return &InputError{
			Status: http.StatusBadRequest,
			Label:  "invalid_json",
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return &InputError{
			Status: http.StatusBadRequest,
			Label:  "invalid_json",
		}
	}

	// 4. Ensure there is no second JSON value or trailing junk.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return &InputError{
			Status: http.StatusBadRequest,
			Label:  "invalid_json",
		}
	}

	// 5. Return without writing a response or mutating application state.
	return nil
}

type InputError struct {
	Status int
	Label  string
}

func (e *InputError) Error() string { return e.Label }
func RejectInput(w http.ResponseWriter, e error) {
	var in *InputError
	if errors.As(e, &in) {
		ErrorJSON(w, in.Status, in.Label)
		return
	}
	ErrorJSON(w, 400, "invalid_input")
}

func StatusResource(s *state.Status) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// GET returns the current status without changing the state.
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": s.Get(),
			})

		case http.MethodPost:
			// Decode the request into a local value for this request.
			// DecodeOne validates the JSON input before the state is changed.
			var request struct {
				Status string `json:"status"`
			}

			if err := DecodeOne(w, r, &request); err != nil {
				// Return the appropriate input error without changing the state.
				RejectInput(w, err)
				return
			}

			// Let the supplied state object validate and apply the new status.
			if err := s.Set(request.Status); err != nil {
				ErrorJSON(w, http.StatusBadRequest, "invalid_status")
				return
			}

			// Return the current status after the update succeeds.
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": s.Get(),
			})

		default:
			// Reject unsupported methods and state which methods are allowed.
			w.Header().Set("Allow", "GET, POST")
			ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
		}
	}
}

// Preserve imports while student bodies are replaced. These declarations are not solutions.
var _ = bytes.NewReader
var _ = json.NewDecoder
var _ = io.EOF
var _ = mime.ParseMediaType

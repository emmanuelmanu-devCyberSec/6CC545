package httpapi

import (
	"encoding/json"
	"net/http"
)

// WriteJSON is provided: it sets headers before the status and writes JSON.
func WriteJSON(w http.ResponseWriter, code int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "response encoding failed", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_, _ = w.Write(append(data, '\n'))
}
func ErrorJSON(w http.ResponseWriter, code int, label string) {
	WriteJSON(w, code, map[string]string{"error": label})
}

func Health(nodeID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{
			"node_id": nodeID,
			"status":  "ok",
		})
	}
}

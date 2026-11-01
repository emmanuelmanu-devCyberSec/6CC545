package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSmoke_JSONHelper(t *testing.T) {
	r := httptest.NewRecorder()
	WriteJSON(r, 200, map[string]string{"hello": "world"})
	if r.Code != 200 || !json.Valid(r.Body.Bytes()) {
		t.Fatal("provided JSON helper failed")
	}
}
func TestWeek01_Health(t *testing.T) {
	for _, id := range []string{"node-a", "node-b"} {
		rr := httptest.NewRecorder()
		Health(id).ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
		var got struct {
			Status string `json:"status"`
			NodeID string `json:"node_id"`
		}
		if e := json.Unmarshal(rr.Body.Bytes(), &got); e != nil {
			t.Fatal(e)
		}
		if rr.Code != 200 || got.Status != "ok" || got.NodeID != id {
			t.Fatalf("incorrect health: %d %s", rr.Code, rr.Body.String())
		}
		if !strings.HasPrefix(rr.Header().Get("Content-Type"), "application/json") {
			t.Fatal("missing JSON content type")
		}
	}
}
func TestWeek01_HealthWrongMethod(t *testing.T) {
	rr := httptest.NewRecorder()
	Health("node-a").ServeHTTP(rr, httptest.NewRequest("POST", "/healthz", nil))
	if rr.Code != 405 {
		t.Fatalf("want 405 got %d", rr.Code)
	}
}

func TestWeek01_HealthWrongMethodErrorJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	Health("node-a").ServeHTTP(rr, httptest.NewRequest("POST", "/healthz", nil))

	var got struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if got.Error != "method not allowed" {
		t.Fatalf("want error %q, got %q", "method not allowed", got.Error)
	}

	if !strings.HasPrefix(rr.Header().Get("Content-Type"), "application/json") {
		t.Fatal("missing JSON content type")
	}
}

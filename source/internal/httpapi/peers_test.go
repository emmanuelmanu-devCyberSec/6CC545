package httpapi

import (
	"example.edu/6cc545/cw1/internal/state"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeek03_PeerAPI(t *testing.T) {
	h := Peers(state.NewRegistry())
	body := `{"id":"node-b","http_address":"127.0.0.1:8082","grpc_address":"127.0.0.1:9092","status":"unknown"}`
	for _, want := range []int{201, 409} {
		r := httptest.NewRequest("POST", "/v1/peers", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("want %d got %d %s", want, w.Code, w.Body.String())
		}
	}
}

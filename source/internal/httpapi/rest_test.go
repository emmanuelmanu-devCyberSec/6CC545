package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"example.edu/6cc545/cw1/internal/state"
)

func TestWeek02_REST(t *testing.T) {
	s := state.NewStatus()
	h := StatusResource(s)
	cases := []struct {
		name, method, body, media string
		code                      int
	}{
		{"get", "GET", "", "", 200}, {"valid", "POST", `{"status":"degraded"}`, "application/json", 200},
		{"charset", "POST", `{"status":"healthy"}`, "application/json; charset=utf-8", 200},
		{"malformed", "POST", `{"status":`, "application/json", 400},
		{"unknown", "POST", `{"status":"healthy","unexpected":1}`, "application/json", 400},
		{"invalid value", "POST", `{"status":"broken"}`, "application/json", 400},
		{"two objects", "POST", `{"status":"healthy"}{}`, "application/json", 400},
		{"null", "POST", `null`, "application/json", 400}, {"array", "POST", `[]`, "application/json", 400},
		{"missing media", "POST", `{}`, "", 415}, {"wrong media", "POST", `{}`, "text/plain", 415},
		{"oversize", "POST", `{"status":"healthy"}` + strings.Repeat(" ", MaxBody), "application/json", 413},
		{"method", "DELETE", "", "", 405},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := s.Get()
			rr := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, "/v1/status", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.media)
			h.ServeHTTP(rr, r)
			if rr.Code != tc.code {
				t.Fatalf("want %d got %d %s", tc.code, rr.Code, rr.Body.String())
			}
			if tc.code >= 400 && s.Get() != before {
				t.Fatal("rejected input changed state")
			}
		})
	}
}

func TestWeek02_REST_TrailingJunkDoesNotMutateState(t *testing.T) {
	// Start with a known state before sending the rejected request.
	s := state.NewStatus()
	if err := s.Set("healthy"); err != nil {
		t.Fatal(err)
	}

	h := StatusResource(s)

	// The JSON object is valid, but the trailing junk makes the
	// complete request body invalid.
	body := `{"status":"degraded"}x`
	r := httptest.NewRequest("POST", "/v1/status", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	// Record the state and request before handling the request.
	t.Logf("BEFORE request: status=%q", s.Get())
	t.Logf("REQUEST: POST /v1/status body=%q", body)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)

	// Record the response and state after handling the request.
	t.Logf("RESPONSE: HTTP %d body=%s", rr.Code, rr.Body.String())
	t.Logf("AFTER request: status=%q", s.Get())

	// The request must be rejected.
	if rr.Code != 400 {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}

	// Rejected input must not change the previously known state.
	if got := s.Get(); got != "healthy" {
		t.Fatalf("rejected request changed state: want healthy, got %q", got)
	}
}

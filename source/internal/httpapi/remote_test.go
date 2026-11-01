package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.edu/6cc545/cw1/internal/model"
	"example.edu/6cc545/cw1/internal/state"
)

// TestWeek05_RemoteFilesUnavailableMapping verifies that the HTTP adapter
// maps an unavailable remote gRPC service to HTTP 503 Service Unavailable.
func TestWeek05_RemoteFilesUnavailableMapping(t *testing.T) {
	// Create a fresh registry for this test.
	reg := state.NewRegistry()

	// Register node-b with a gRPC address where no service is listening.
	// This makes the remote request fail without requiring another test server.
	err := reg.Add(model.Peer{
		ID:          "node-b",
		HTTPAddress: "127.0.0.1:8082",
		GRPCAddress: "127.0.0.1:1",
		Status:      "unknown",
	})
	if err != nil {
		t.Fatalf("register peer: %v", err)
	}

	// Create the remote-files HTTP handler with a bounded RPC timeout.
	handler := RemoteFiles(reg, "node-a", time.Second)

	// Build a request asking node-a for node-b's remote file metadata.
	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/remote-files?peer_id=node-b",
		nil,
	)
	rec := httptest.NewRecorder()

	// Execute the request through the HTTP adapter.
	handler.ServeHTTP(rec, req)

	// Verify that the unavailable gRPC service is mapped to HTTP 503.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want HTTP 503, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify that the response contains the expected JSON error identifier.
	if !strings.Contains(rec.Body.String(), "remote_unavailable") {
		t.Fatalf("expected remote_unavailable error, got %s", rec.Body.String())
	}
}

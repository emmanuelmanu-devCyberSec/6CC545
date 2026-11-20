//go:build acceptance

package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"example.edu/6cc545/cw1/internal/config"
)

func TestRequiredSecurityModeRejectsIncompleteConfiguration(t *testing.T) {
	c := config.Config{
		ID:                "node-a",
		HTTP:              "127.0.0.1:18081",
		DataDir:           t.TempDir(),
		Timeout:           2 * time.Second,
		SecurityMode:      "required",
		PeerHTTPAddr:      "127.0.0.1:19444",
		PeerHTTPAdvertise: "https://node-a:9444",
	}

	err := Run(context.Background(), c)
	if err == nil {
		t.Fatal("expected startup to fail when required JWT configuration is missing")
	}

	if !strings.Contains(err.Error(), "incomplete JWT configuration") {
		t.Fatalf("expected incomplete JWT configuration error, got: %v", err)
	}
}

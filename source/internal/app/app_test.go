package app

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"example.edu/6cc545/cw1/internal/config"
)

// TestWeek01_OccupiedPort checks that the application reports a clear error
// when another process is already using the configured HTTP listening port.
func TestWeek01_OccupiedPort(t *testing.T) {
	// Reserve an available local port so the application cannot bind to it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Give the node a temporary storage directory to avoid writing test data
	// into the project's actual data directory.
	c := config.Config{
		ID:      "node-a",
		HTTP:    ln.Addr().String(),
		DataDir: t.TempDir(),
		Timeout: time.Second,
	}

	// Limit the test duration in case application startup does not finish.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the application and check that the occupied port is reported.
	err = Run(ctx, c)
	if err == nil || !strings.Contains(err.Error(), "listen HTTP") {
		t.Fatalf("want clear bind error, got %v", err)
	}
}

// TestWeek01_CancelledLifecycle checks that the application handles
// an already-cancelled context without returning an error.
func TestWeek01_CancelledLifecycle(t *testing.T) {
	// Reserve a local port to obtain a valid HTTP listening address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	address := ln.Addr().String()

	// Release the port before starting the application.
	_ = ln.Close()

	// Create and immediately cancel the application context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use an isolated temporary directory for this test's storage.
	c := config.Config{
		ID:      "node-a",
		HTTP:    address,
		DataDir: t.TempDir(),
		Timeout: time.Second,
	}

	// Verify that startup and shutdown handle the cancelled context correctly.
	if err = Run(ctx, c); err != nil {
		t.Fatal(err)
	}
}

// TestWeek08_ActiveLifecycleShutdown checks that a running node
// shuts down successfully when its application context is cancelled.
func TestWeek08_ActiveLifecycleShutdown(t *testing.T) {
	// Reserve a local port to obtain an available HTTP address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	address := ln.Addr().String()
	_ = ln.Close()

	// Create a cancellable context for the running application.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a temporary directory so the test does not affect real node data.
	c := config.Config{
		ID:      "node-shutdown-test",
		HTTP:    address,
		DataDir: t.TempDir(),
		Timeout: time.Second,
	}

	// Run the application in the background.
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, c)
	}()

	// Use a short HTTP timeout while waiting for the health endpoint.
	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	startupDeadline := time.Now().Add(2 * time.Second)

	// Wait until the node is responding to health checks.
	for {
		resp, requestErr := client.Get("http://" + address + "/healthz")
		if requestErr == nil {
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				cancel()
				<-done
				t.Fatalf("expected HTTP 200 from health endpoint, got %d", resp.StatusCode)
			}

			break
		}

		// Detect an application error before the health endpoint becomes ready.
		select {
		case runErr := <-done:
			t.Fatalf("application stopped before becoming ready: %v", runErr)
		default:
		}

		if time.Now().After(startupDeadline) {
			cancel()
			<-done
			t.Fatal("application did not become ready within the startup deadline")
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Cancel the application after confirming that it is running.
	cancel()

	// Shutdown must complete within the bounded test period.
	select {
	case runErr := <-done:
		if runErr != nil {
			t.Fatalf("application shutdown returned an error: %v", runErr)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("application did not shut down within the expected period")
	}
}

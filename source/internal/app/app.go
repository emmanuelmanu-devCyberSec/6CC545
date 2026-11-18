package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/httpapi"
)

// Run starts the node and owns its HTTP server and application resources.
// When the application context is cancelled, it performs bounded cleanup.
func Run(ctx context.Context, c config.Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := CheckStage(c); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Exit cleanly if the application context was already cancelled.
	if ctx.Err() != nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpapi.ErrorJSON(w, http.StatusNotFound, "not_found")
	})
	mux.HandleFunc("/healthz", httpapi.Health(c.ID))

	// Wire creates the node's internal services and returns their cleanup function.
	stop, err := Wire(ctx, c, mux)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", c.HTTP)
	if err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			c.Timeout,
		)
		defer shutdownCancel()

		if cleanupErr := stop(shutdownCtx); cleanupErr != nil {
			return errors.Join(
				fmt.Errorf("listen HTTP %s: %w", c.HTTP, err),
				fmt.Errorf("application cleanup: %w", cleanupErr),
			)
		}

		return fmt.Errorf("listen HTTP %s: %w", c.HTTP, err)
	}

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: c.Timeout,
		ReadTimeout:       2 * c.Timeout,
		WriteTimeout:      2 * c.Timeout,
		IdleTimeout:       30 * time.Second,
	}

	done := make(chan error, 1)

	// Serve runs in its own goroutine so Run can respond to cancellation.
	go func() {
		done <- srv.Serve(ln)
	}()

	log.Printf(
		"node=%s http=%s ready (loopback teaching service; no authentication)",
		c.ID,
		ln.Addr(),
	)

	var runErr error

	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}

	case <-ctx.Done():
		// The application context is cancelled, so use a fresh context
		// to give HTTP requests and the other services time to shut down.
	}

	// Use one shared deadline for HTTP shutdown and all application cleanup.
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		c.Timeout,
	)
	defer shutdownCancel()

	// Stop accepting new HTTP requests and allow active requests to finish.
	shutdownErr := srv.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		// Force-close remaining connections if graceful shutdown times out.
		_ = srv.Close()
	}

	// Wait for the HTTP Serve goroutine to finish without waiting forever.
	select {
	case serveErr := <-done:
		if !errors.Is(serveErr, http.ErrServerClosed) && runErr == nil {
			runErr = serveErr
		}
	case <-shutdownCtx.Done():
		if runErr == nil {
			runErr = fmt.Errorf("HTTP server did not stop before shutdown deadline")
		}
	}

	// Stop internal services using the same shutdown deadline.
	cleanupErr := stop(shutdownCtx)

	return errors.Join(
		runErr,
		wrapShutdownError("HTTP shutdown", shutdownErr),
		wrapShutdownError("application cleanup", cleanupErr),
	)
}

// wrapShutdownError adds context to a shutdown error without changing nil values.
func wrapShutdownError(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}

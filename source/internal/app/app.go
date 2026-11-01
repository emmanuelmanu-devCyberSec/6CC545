package app

import (
	"context"
	"errors"
	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/httpapi"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// Run is a supplied lifecycle example. Explain its ownership and cleanup decisions.
func Run(ctx context.Context, c config.Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := CheckStage(c); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { httpapi.ErrorJSON(w, 404, "not_found") })
	mux.HandleFunc("/healthz", httpapi.Health(c.ID))
	stop, err := Wire(ctx, c, mux)
	if err != nil {
		return err
	}
	defer stop()
	ln, err := net.Listen("tcp", c.HTTP)
	if err != nil {
		return fmt.Errorf("listen HTTP %s: %w", c.HTTP, err)
	}
	defer ln.Close()
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: c.Timeout, ReadTimeout: 2 * c.Timeout, WriteTimeout: 2 * c.Timeout, IdleTimeout: 30 * time.Second}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	log.Printf("node=%s http=%s ready (loopback teaching service; no authentication)", c.ID, ln.Addr())
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// Use a fresh context: the application context is already cancelled.
		shutdownCtx, closeCtx := context.WithTimeout(context.Background(), c.Timeout)
		defer closeCtx()
		shutdownErr := srv.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			_ = srv.Close()
		}
		<-done
		if shutdownErr != nil {
			return fmt.Errorf("HTTP shutdown: %w", shutdownErr)
		}
		return nil
	}
}

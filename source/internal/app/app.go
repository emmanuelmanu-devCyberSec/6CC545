package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/httpapi"
	"example.edu/6cc545/cw1/internal/security"
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

	if ctx.Err() != nil {
		return nil
	}

	// Prepare HTTPS before opening the listener when security is required.
	var httpTLSConfig *tls.Config
	if c.SecurityMode == "required" {
		var err error
		httpTLSConfig, err = security.ServerTLS(
			c.TLSCertFile,
			c.TLSKeyFile,
			c.CAFile,
			false,
		)
		if err != nil {
			return fmt.Errorf("configure HTTPS: %w", err)
		}
	}

	// Bind the HTTP port first so an occupied port is reported
	// before application components are initialized.
	ln, err := net.Listen("tcp", c.HTTP)
	if err != nil {
		return fmt.Errorf("listen HTTP %s: %w", c.HTTP, err)
	}

	// Required security mode serves the API through TLS.
	// The legacy HTTP path is retained only for existing health-only tests.
	serveListener := ln
	protocol := "http"

	if httpTLSConfig != nil {
		serveListener = tls.NewListener(ln, httpTLSConfig)
		protocol = "https"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpapi.ErrorJSON(w, http.StatusNotFound, "not_found")
	})
	mux.HandleFunc("/healthz", httpapi.Health(c.ID))

	var stop func(context.Context) error

	if c.JWTKeyFile == "" && c.JWTIssuer == "" && c.JWTAudience == "" {
		// Legacy lifecycle tests can start the public health endpoint
		// without enabling the protected application services.
		log.Printf("node=%s starting in health-only mode: JWT configuration is absent", c.ID)

		stop = func(context.Context) error {
			return nil
		}
	} else {
		stop, err = Wire(ctx, c, mux)
		if err != nil {
			_ = ln.Close()
			return err
		}
	}

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: c.Timeout,
		ReadTimeout:       2 * c.Timeout,
		WriteTimeout:      2 * c.Timeout,
		IdleTimeout:       30 * time.Second,
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(serveListener)
	}()

	log.Printf("node=%s protocol=%s address=%s ready", c.ID, protocol, ln.Addr())

	var runErr error

	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		c.Timeout,
	)
	defer shutdownCancel()

	shutdownErr := srv.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		_ = srv.Close()
	}

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

// deploycheck is SUPPLIED deployment infrastructure, not the student's CW2 node.
// Replace APP_PACKAGE with ./cmd/node after completing the integration checklist.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func main() {
	id := env("NODE_ID", "deploycheck")
	addr := env("HTTP_ADDR", "127.0.0.1:8080")
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"node_id": id, "status": "ok", "mode": "deployment-scaffold-only", "assessment_ready": false})
	})
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 2 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 15 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Printf("node=%s mode=deployment-scaffold-only address=%s", id, addr)
	select {
	case e := <-done:
		if !errors.Is(e, http.ErrServerClosed) {
			log.Fatal(e)
		}
	case <-ctx.Done():
		stop, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		if e := server.Shutdown(stop); e != nil {
			_ = server.Close()
		}
		<-done
	}
}

package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/httpapi"
	"example.edu/6cc545/cw1/internal/ipc"
	"example.edu/6cc545/cw1/internal/model"
	"example.edu/6cc545/cw1/internal/rpcfiles"
	"example.edu/6cc545/cw1/internal/state"
	"google.golang.org/grpc"
)

func loadSeed(c *state.Catalog, path string) error {
	if path == "" {
		return nil
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil {
		return e
	}
	if len(b) > 65536 {
		return errors.New("seed exceeds 64 KiB")
	}
	var data []model.FileMetadata
	if e = json.Unmarshal(b, &data); e != nil {
		return e
	}
	return c.Replace(data)
}
func base(c config.Config, mux *http.ServeMux) (*state.Registry, *state.Catalog, error) {
	reg := state.NewRegistry()
	cat := state.NewCatalog()
	if e := loadSeed(cat, c.Seed); e != nil {
		return nil, nil, e
	}
	mux.HandleFunc("/v1/status", httpapi.StatusResource(state.NewStatus()))
	mux.HandleFunc("/v1/peers", httpapi.Peers(reg))
	mux.HandleFunc("/v1/files", httpapi.Files(cat))
	return reg, cat, nil
}

// Socket setup/ownership is supplied; core protocol functions are student work.
func wireIPC(ctx context.Context, c config.Config, mux *http.ServeMux) (func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	var closers []io.Closer
	inbox := state.NewInbox()
	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			httpapi.ErrorJSON(w, 405, "method_not_allowed")
			return
		}
		httpapi.WriteJSON(w, 200, inbox.Snapshot())
	})
	cleanup := func() {
		cancel()
		for _, x := range closers {
			_ = x.Close()
		}
		wg.Wait()
	}
	if c.TCP != "" {
		ln, e := net.Listen("tcp", c.TCP)
		if e != nil {
			cleanup()
			return nil, e
		}
		closers = append(closers, ln)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := ipc.ServeTCP(ctx, ln, c.ID, c.Timeout, inbox.Record); e != nil && ctx.Err() == nil {
				log.Printf("TCP service stopped: %v", e)
			}
		}()
	}
	if c.UDP != "" {
		a, e := net.ResolveUDPAddr("udp", c.UDP)
		if e != nil {
			cleanup()
			return nil, e
		}
		s, e := net.ListenUDP("udp", a)
		if e != nil {
			cleanup()
			return nil, e
		}
		closers = append(closers, s)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := ipc.ServeUDP(ctx, s, c.ID, c.Timeout, inbox.Record); e != nil && ctx.Err() == nil {
				log.Printf("UDP service stopped: %v", e)
			}
		}()
	}
	return cleanup, nil
}

func CheckStage(c config.Config) error { return nil }
func Wire(ctx context.Context, c config.Config, mux *http.ServeMux) (func(), error) {
	reg, cat, e := base(c, mux)
	if e != nil {
		return nil, e
	}
	stopIPC, e := wireIPC(ctx, c, mux)
	if e != nil {
		return nil, e
	}
	mux.HandleFunc("/v1/remote-files", httpapi.RemoteFiles(reg, c.ID, c.Timeout))
	if c.GRPC == "" {
		return stopIPC, nil
	}
	ln, e := net.Listen("tcp", c.GRPC)
	if e != nil {
		stopIPC()
		return nil, e
	}
	s := grpc.NewServer(grpc.MaxRecvMsgSize(4096), grpc.MaxSendMsgSize(65536))
	pb.RegisterFileServiceServer(s, &rpcfiles.Service{NodeID: c.ID, Catalog: cat})
	done := make(chan struct{})
	go func() {
		defer close(done)
		if e := s.Serve(ln); e != nil && ctx.Err() == nil {
			log.Printf("gRPC stopped: %v", e)
		}
	}()
	return func() {
		stopped := make(chan struct{})
		go func() { s.GracefulStop(); close(stopped) }()
		select {
		case <-stopped:
		case <-time.After(c.Timeout):
			s.Stop()
			<-stopped
		}
		_ = ln.Close()
		<-done
		stopIPC()
	}, nil
}

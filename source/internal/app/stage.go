package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	pb "example.edu/6cc545/cw1/api/files/v1"
	"example.edu/6cc545/cw1/internal/config"
	"example.edu/6cc545/cw1/internal/events"
	"example.edu/6cc545/cw1/internal/httpapi"
	"example.edu/6cc545/cw1/internal/ipc"
	"example.edu/6cc545/cw1/internal/model"
	"example.edu/6cc545/cw1/internal/rpcfiles"
	"example.edu/6cc545/cw1/internal/state"
	"example.edu/6cc545/cw1/internal/storage"
	"google.golang.org/grpc"
)

// startFileAddedConsumers attaches two independent consumers to file.added events.
// The counter tracks received metadata events, while the logger records
// structured diagnostic information without accessing file contents.
func startFileAddedConsumers(ctx context.Context, bus *events.Bus) (func(), error) {
	counterSub, err := bus.Subscribe("file.added")
	if err != nil {
		return nil, err
	}

	loggerSub, err := bus.Subscribe("file.added")
	if err != nil {
		counterSub.Cancel()
		return nil, err
	}

	var count atomic.Uint64
	var wg sync.WaitGroup
	var stopOnce sync.Once

	consumerCtx, cancel := context.WithCancel(ctx)

	// Consumer 1: count each received file metadata event.
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-consumerCtx.Done():
				return
			case event, ok := <-counterSub.C:
				if !ok {
					return
				}

				total := count.Add(1)
				log.Printf(
					"event_counter count=%d event_id=%q name=%q",
					total, event.ID, event.Name,
				)
			}
		}
	}()

	// Consumer 2: log structured event metadata for diagnostics.
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-consumerCtx.Done():
				return
			case event, ok := <-loggerSub.C:
				if !ok {
					return
				}

				log.Printf(
					"event_received id=%q kind=%q origin=%q name=%q",
					event.ID, event.Kind, event.Origin, event.Name,
				)
			}
		}
	}()

	// Stop both consumers and release their subscriptions safely.
	stop := func() {
		stopOnce.Do(func() {
			cancel()
			counterSub.Cancel()
			loggerSub.Cancel()
			wg.Wait()
		})
	}

	return stop, nil
}

// loadSeed registers the seed metadata and publishes one local event
// for each registered record. The forwarding worker must be running
// before this function is called so it can receive these events.
func loadSeed(c *state.Catalog, path string, bus *events.Bus, origin string) error {
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

	// Register all metadata before publishing any file.added events.
	if e = c.Replace(data); e != nil {
		return e
	}

	// Publish one metadata event for each successfully registered record.
	for i, file := range data {
		event := events.Event{
			ID:     fmt.Sprintf("%s-file-%d", origin, i+1),
			Kind:   "file.added",
			Origin: origin,
			Name:   file.Name,
		}

		if _, e = bus.Publish(event); e != nil {
			return fmt.Errorf("publish file metadata event: %w", e)
		}
	}

	return nil
}

// forwardFileAdded converts a local file.added event into an IPC notification
// and sends it to registered peers through their HTTP message endpoints.
// The event ID and original node ID are preserved in the IPC message.
func forwardFileAdded(
	ctx context.Context,
	reg *state.Registry,
	localID string,
	event events.Event,
) error {
	message := ipc.Message{
		Version:   ipc.Version,
		RequestID: event.ID,
		NodeID:    event.Origin,
		Text:      event.Kind + ":" + event.Name,
	}

	if err := message.Validate(); err != nil {
		return fmt.Errorf("invalid file event: %w", err)
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode peer notification: %w", err)
	}

	peers := reg.Snapshot()
	sent := 0
	var firstErr error

	for _, peer := range peers {
		if peer.ID == localID {
			continue
		}

		if peer.Status == "offline" {
			continue
		}

		url := "http://" + peer.HTTPAddress + "/v1/messages"

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			url,
			bytes.NewReader(payload),
		)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"create notification request for %s: %w",
					peer.ID,
					err,
				)
			}
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"send notification to %s: %w",
					peer.ID,
					err,
				)
			}
			continue
		}

		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()

		if resp.StatusCode < http.StatusOK ||
			resp.StatusCode >= http.StatusMultipleChoices {
			if firstErr == nil {
				firstErr = fmt.Errorf(
					"peer %s returned HTTP status %d",
					peer.ID,
					resp.StatusCode,
				)
			}
			continue
		}

		sent++
	}

	if sent == 0 && firstErr == nil {
		log.Printf(
			"event_forward_no_peer event_id=%q origin=%q",
			event.ID,
			event.Origin,
		)
	}

	return firstErr
}

// base creates the peer registry, catalogue and HTTP endpoints.
// The Store handles file contents; the catalogue holds committed metadata.
func base(
	c config.Config,
	mux *http.ServeMux,
	bus *events.Bus,
	store *storage.Store,
	maxUploadBytes int64,
) (*state.Registry, *state.Catalog, error) {
	reg := state.NewRegistry()
	cat := state.NewCatalog()

	mux.HandleFunc("/v1/status", httpapi.StatusResource(state.NewStatus()))
	mux.HandleFunc("/v1/peers", httpapi.Peers(reg))

	// Generate a unique ID for each file event.
	var fileEventSequence atomic.Uint64

	// Publish only metadata for a file that has already been committed.
	publishFileEvent := func(
		file model.FileMetadata,
	) (string, events.Delivery, error) {
		eventID := fmt.Sprintf(
			"new-file-%d",
			fileEventSequence.Add(1),
		)

		event := events.Event{
			ID:     eventID,
			Kind:   "file.added",
			Origin: c.ID,
			Name:   file.Name,
		}

		// Return the actual delivery counts to the HTTP handler.
		delivery, err := bus.Publish(event)

		return eventID, delivery, err
	}

	// POST uploads a file through Store.Put.
	// GET returns the committed file catalogue.
	mux.HandleFunc("/v1/files", httpapi.Files(
		cat,
		store,
		maxUploadBytes,
		publishFileEvent,
	))

	return reg, cat, nil
}

// Socket setup/ownership is supplied; core protocol functions are student work.
func wireIPC(ctx context.Context, c config.Config, mux *http.ServeMux) (func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	var closers []io.Closer

	inbox := state.NewInbox()

	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			httpapi.WriteJSON(w, http.StatusOK, inbox.Snapshot())

		case http.MethodPost:
			var message ipc.Message
			if err := httpapi.DecodeOne(w, r, &message); err != nil {
				httpapi.RejectInput(w, err)
				return
			}

			if err := message.Validate(); err != nil {
				httpapi.ErrorJSON(w, http.StatusBadRequest, "invalid_message")
				return
			}

			added := inbox.RecordIfNew(message)

			httpapi.WriteJSON(w, http.StatusAccepted, map[string]bool{
				"accepted":  true,
				"duplicate": !added,
			})

		default:
			httpapi.ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
		}
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
	// Create an event bus with bounded queues for local application events.
	bus, e := events.New(16)
	if e != nil {
		return nil, e
	}

	// Start both local consumers before loading the seed metadata.
	stopConsumers, e := startFileAddedConsumers(ctx, bus)
	if e != nil {
		_ = bus.Close()
		return nil, e
	}

	// Stop consumers before closing the event bus.
	cleanupEvents := func() {
		stopConsumers()
		_ = bus.Close()
	}

	// Create the local file store.
	// Each node has its own directory and a maximum upload size of 10 MiB.
	const maxUploadBytes int64 = 10 << 20

	store, e := storage.New(
		filepath.Join("data", c.ID),
		maxUploadBytes,
	)
	if e != nil {
		cleanupEvents()
		return nil, fmt.Errorf("create file store: %w", e)
	}

	// Create the registry and catalogue and connect the Store to the HTTP route.
	reg, cat, e := base(c, mux, bus, store, maxUploadBytes)
	if e != nil {
		cleanupEvents()
		return nil, e
	}

	// The application owns the forwarding worker and waits for it at shutdown.
	forwardCtx, cancelForward := context.WithCancel(ctx)
	var forwardWG sync.WaitGroup

	forwardWG.Add(1)
	go func() {
		defer forwardWG.Done()

		err := bus.Forward(
			forwardCtx,
			"file.added",
			func(sendCtx context.Context, event events.Event) error {
				return forwardFileAdded(sendCtx, reg, c.ID, event)
			},
		)

		if err != nil && forwardCtx.Err() == nil {
			log.Printf("peer forwarding stopped: %v", err)
		}
	}()

	stopForwarder := func() {
		cancelForward()
		forwardWG.Wait()
	}

	// Publish seed events after starting the forwarding worker.
	if e = loadSeed(cat, c.Seed, bus, c.ID); e != nil {
		stopForwarder()
		cleanupEvents()
		return nil, e
	}

	stopIPC, e := wireIPC(ctx, c, mux)
	if e != nil {
		stopForwarder()
		cleanupEvents()
		return nil, e
	}

	mux.HandleFunc("/v1/remote-files", httpapi.RemoteFiles(reg, c.ID, c.Timeout))

	if c.GRPC == "" {
		return func() {
			stopIPC()
			stopForwarder()
			cleanupEvents()
		}, nil
	}

	ln, e := net.Listen("tcp", c.GRPC)
	if e != nil {
		stopIPC()
		stopForwarder()
		cleanupEvents()
		return nil, e
	}

	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(4096),
		grpc.MaxSendMsgSize(65536),
	)
	pb.RegisterFileServiceServer(s, &rpcfiles.Service{
		NodeID:  c.ID,
		Catalog: cat,
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if e := s.Serve(ln); e != nil && ctx.Err() == nil {
			log.Printf("gRPC stopped: %v", e)
		}
	}()

	return func() {
		stopped := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
		case <-time.After(c.Timeout):
			s.Stop()
			<-stopped
		}

		_ = ln.Close()
		<-done

		stopIPC()
		stopForwarder()
		cleanupEvents()
	}, nil
}

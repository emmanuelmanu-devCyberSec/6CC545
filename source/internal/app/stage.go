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
	// Convert the local event into the existing IPC message format.
	message := ipc.Message{
		Version:   ipc.Version,
		RequestID: event.ID,
		NodeID:    event.Origin,
		Text:      event.Kind + ":" + event.Name,
	}

	// Validate the message before attempting network transmission.
	if err := message.Validate(); err != nil {
		return fmt.Errorf("invalid file event: %w", err)
	}

	// Encode notification metadata only. File contents are not transmitted.
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode peer notification: %w", err)
	}

	// Read the current peer registry so newly registered peers are included.
	peers := reg.Snapshot()
	sent := 0
	var firstErr error

	for _, peer := range peers {
		// Do not send an event back to its originating node.
		if peer.ID == localID {
			continue
		}

		// Do not attempt delivery to peers explicitly marked offline.
		if peer.Status == "offline" {
			continue
		}

		// Use the registered HTTP address for notification delivery.
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

		// Bus.Forward supplies a context with a finite per-send deadline.
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

		// Close the response body after checking the receiver's acknowledgement.
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

	// A missing peer is not a delivery failure, but it is recorded for diagnostics.
	if sent == 0 && firstErr == nil {
		log.Printf(
			"event_forward_no_peer event_id=%q origin=%q",
			event.ID,
			event.Origin,
		)
	}

	return firstErr
}

// base creates the peer registry, metadata catalogue and HTTP endpoints.
// Seed loading is handled later in Wire, after the forwarding worker starts.
func base(
	c config.Config,
	mux *http.ServeMux,
	bus *events.Bus,
) (*state.Registry, *state.Catalog, error) {
	reg := state.NewRegistry()
	cat := state.NewCatalog()

	// Register the application endpoints.
	mux.HandleFunc("/v1/status", httpapi.StatusResource(state.NewStatus()))
	mux.HandleFunc("/v1/peers", httpapi.Peers(reg))
	// Create a sequence for uniquely identifying manually added file events.
	var fileEventSequence atomic.Uint64

	// Publish a file.added event whenever new metadata is added to the catalogue.
	publishFileEvent := func(file model.FileMetadata) (string, error) {
		// Generate an event ID for this notification.
		eventID := fmt.Sprintf("new-file-%d", fileEventSequence.Add(1))

		// Build the event using the local node as its origin.
		event := events.Event{
			ID:     eventID,
			Kind:   "file.added",
			Origin: c.ID,
			Name:   file.Name,
		}

		// Publish the event to the local event bus.
		if _, err := bus.Publish(event); err != nil {
			return "", err
		}

		// Return the event ID so the HTTP handler can include it in its response.
		return eventID, nil
	}

	// Register the files endpoint with support for listing and adding metadata.
	mux.HandleFunc("/v1/files", httpapi.Files(cat, publishFileEvent))

	return reg, cat, nil
}

// Socket setup/ownership is supplied; core protocol functions are student work.
func wireIPC(ctx context.Context, c config.Config, mux *http.ServeMux) (func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	var closers []io.Closer

	inbox := state.NewInbox()

	// Handle both reading and receiving peer notifications over HTTP.
	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Return the messages currently held in the local inbox.
			httpapi.WriteJSON(w, http.StatusOK, inbox.Snapshot())

		case http.MethodPost:
			// Decode and validate the incoming notification.
			var message ipc.Message
			if err := httpapi.DecodeOne(w, r, &message); err != nil {
				httpapi.RejectInput(w, err)
				return
			}

			// Check the IPC message fields before recording the notification.
			if err := message.Validate(); err != nil {
				httpapi.ErrorJSON(w, http.StatusBadRequest, "invalid_message")
				return
			}

			// Record only new notifications. A repeated notification from
			// the same origin with the same request ID is treated as a duplicate.
			added := inbox.RecordIfNew(message)

			// Acknowledge receipt without claiming that further processing
			// or metadata reconciliation has been completed.
			httpapi.WriteJSON(w, http.StatusAccepted, map[string]bool{
				"accepted":  true,
				"duplicate": !added,
			})

		default:
			// Only GET and POST are supported by this endpoint.
			httpapi.ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed")
		}
	})

	cleanup := func() {
		// Cancel active socket services before closing their listeners.
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

	// Create the registry and catalogue before starting peer forwarding.
	reg, cat, e := base(c, mux, bus)
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

		// Subscribe once and forward file.added events to registered peers.
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

	// Cancel the forwarding worker and wait for it to finish before
	// the event bus is closed.
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

	// Expose the existing gRPC metadata-fetching endpoint.
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
		// Gracefully stop the gRPC server, with a timeout fallback.
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

		// Stop socket services and the forwarding worker before
		// stopping local consumers and closing the event bus.
		stopIPC()
		stopForwarder()
		cleanupEvents()
	}, nil
}

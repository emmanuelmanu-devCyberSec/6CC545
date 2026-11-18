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
func startFileAddedConsumers(
	ctx context.Context,
	bus *events.Bus,
) (func(context.Context) error, error) {
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

	// Cancel both consumers and release their subscriptions safely.
	stop := func(shutdownCtx context.Context) error {
		stopOnce.Do(func() {
			cancel()
			counterSub.Cancel()
			loggerSub.Cancel()
		})

		return waitForWorkers(shutdownCtx, &wg)
	}

	return stop, nil
}

// waitForWorkers waits for application-owned goroutines until the
// supplied shutdown context expires.
func waitForWorkers(ctx context.Context, wg *sync.WaitGroup) error {
	done := make(chan struct{})

	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// loadSeed registers the seed metadata and publishes one local event
// for each registered record. The forwarding worker must be running
// before this function is called so it can receive these events.
func loadSeed(c *state.Catalog, path string, bus *events.Bus, origin string) error {
	if path == "" {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return err
	}
	if len(b) > 65536 {
		return errors.New("seed exceeds 64 KiB")
	}

	var data []model.FileMetadata
	if err = json.Unmarshal(b, &data); err != nil {
		return err
	}

	// Register all metadata before publishing any file.added events.
	if err = c.Replace(data); err != nil {
		return err
	}

	// Publish one metadata event for each successfully registered record.
	for i, file := range data {
		event := events.Event{
			ID:     fmt.Sprintf("%s-file-%d", origin, i+1),
			Kind:   "file.added",
			Origin: origin,
			Name:   file.Name,
		}

		if _, err = bus.Publish(event); err != nil {
			return fmt.Errorf("publish file metadata event: %w", err)
		}
	}

	return nil
}

// forwardFileAdded converts a local file.added event into an IPC notification
// and sends it to registered peers through their HTTP message endpoints.
// The event ID and original node ID are preserved in the IPC message.
// Each peer request is limited by the configured network timeout.
func forwardFileAdded(
	ctx context.Context,
	reg *state.Registry,
	localID string,
	event events.Event,
	timeout time.Duration,
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

	// Use a bounded HTTP client so an unresponsive peer cannot hold
	// the forwarding worker indefinitely.
	client := &http.Client{
		Timeout: timeout,
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

		resp, err := client.Do(req)
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

	// Restore committed file metadata from the recovered storage.
	storedFiles, err := store.Snapshot()
	if err != nil {
		return nil, nil, err
	}

	recoveredFiles := make([]model.FileMetadata, 0, len(storedFiles))

	for _, file := range storedFiles {
		info, err := os.Stat(filepath.Join(c.DataDir, file.Name))
		if err != nil {
			return nil, nil, err
		}

		recoveredFiles = append(recoveredFiles, model.FileMetadata{
			Name:         file.Name,
			Size:         file.Size,
			SHA256:       file.SHA256,
			ModifiedUnix: info.ModTime().Unix(),
		})
	}

	if err := cat.Replace(recoveredFiles); err != nil {
		return nil, nil, err
	}

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

// wireIPC starts the node's TCP and UDP services and owns their listeners.
// Its cleanup function cancels the services, closes their listeners and
// waits for their goroutines within the supplied shutdown deadline.
func wireIPC(
	ctx context.Context,
	c config.Config,
	mux *http.ServeMux,
) (func(context.Context) error, error) {
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	var closers []io.Closer
	var closeOnce sync.Once

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

	// Cancel the services and close their listeners before waiting
	// for their goroutines to finish.
	cleanup := func(shutdownCtx context.Context) error {
		cancel()

		closeOnce.Do(func() {
			for _, closer := range closers {
				_ = closer.Close()
			}
		})

		return waitForWorkers(shutdownCtx, &wg)
	}

	// Use a bounded cleanup context if service startup fails partway through.
	cleanupAfterSetupError := func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			c.Timeout,
		)
		defer shutdownCancel()

		if err := cleanup(shutdownCtx); err != nil {
			log.Printf("IPC startup cleanup: %v", err)
		}
	}

	if c.TCP != "" {
		ln, err := net.Listen("tcp", c.TCP)
		if err != nil {
			cleanupAfterSetupError()
			return nil, err
		}

		closers = append(closers, ln)

		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := ipc.ServeTCP(ctx, ln, c.ID, c.Timeout, inbox.Record); err != nil && ctx.Err() == nil {
				log.Printf("TCP service stopped: %v", err)
			}
		}()
	}

	if c.UDP != "" {
		addr, err := net.ResolveUDPAddr("udp", c.UDP)
		if err != nil {
			cleanupAfterSetupError()
			return nil, err
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			cleanupAfterSetupError()
			return nil, err
		}

		closers = append(closers, conn)

		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := ipc.ServeUDP(ctx, conn, c.ID, c.Timeout, inbox.Record); err != nil && ctx.Err() == nil {
				log.Printf("UDP service stopped: %v", err)
			}
		}()
	}

	return cleanup, nil
}

func CheckStage(c config.Config) error { return nil }

func Wire(
	ctx context.Context,
	c config.Config,
	mux *http.ServeMux,
) (func(context.Context) error, error) {
	// Create an event bus with bounded queues for local application events.
	bus, err := events.New(16)
	if err != nil {
		return nil, err
	}

	// Start both local consumers before loading the seed metadata.
	stopConsumers, err := startFileAddedConsumers(ctx, bus)
	if err != nil {
		_ = bus.Close()
		return nil, err
	}

	// Create the local file store using the configured storage directory.
	const maxUploadBytes int64 = 10 << 20

	store, err := storage.New(c.DataDir, maxUploadBytes)
	if err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			c.Timeout,
		)
		defer shutdownCancel()

		_ = stopConsumers(shutdownCtx)
		_ = bus.Close()

		return nil, fmt.Errorf("create file store: %w", err)
	}

	// Recover committed files before registering routes or starting services.
	if err := store.Recover(ctx); err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			c.Timeout,
		)
		defer shutdownCancel()

		_ = stopConsumers(shutdownCtx)
		_ = bus.Close()

		return nil, fmt.Errorf("recover file store: %w", err)
	}

	// Create the registry and catalogue and connect the Store to the HTTP route.
	reg, cat, err := base(c, mux, bus, store, maxUploadBytes)
	if err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			c.Timeout,
		)
		defer shutdownCancel()

		_ = stopConsumers(shutdownCtx)
		_ = bus.Close()

		return nil, err
	}

	// Start the forwarding worker before publishing seed events.
	forwardCtx, cancelForward := context.WithCancel(ctx)
	var forwardWG sync.WaitGroup

	forwardWG.Add(1)
	go func() {
		defer forwardWG.Done()

		err := bus.Forward(
			forwardCtx,
			"file.added",
			func(sendCtx context.Context, event events.Event) error {
				return forwardFileAdded(
					sendCtx,
					reg,
					c.ID,
					event,
					c.Timeout,
				)
			},
		)

		if err != nil && forwardCtx.Err() == nil {
			log.Printf("peer forwarding stopped: %v", err)
		}
	}()

	// Start the IPC services.
	stopIPC, err := wireIPC(ctx, c, mux)
	if err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			c.Timeout,
		)
		defer shutdownCancel()

		cancelForward()
		_ = waitForWorkers(shutdownCtx, &forwardWG)
		_ = stopConsumers(shutdownCtx)
		_ = bus.Close()

		return nil, err
	}

	mux.HandleFunc(
		"/v1/remote-files",
		httpapi.RemoteFiles(reg, c.ID, c.Timeout),
	)

	// Start gRPC only when its listening address is configured.
	var grpcServer *grpc.Server
	var grpcDone chan struct{}

	if c.GRPC != "" {
		ln, listenErr := net.Listen("tcp", c.GRPC)
		if listenErr != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(
				context.Background(),
				c.Timeout,
			)
			defer shutdownCancel()

			_ = stopIPC(shutdownCtx)
			cancelForward()
			_ = waitForWorkers(shutdownCtx, &forwardWG)
			_ = stopConsumers(shutdownCtx)
			_ = bus.Close()

			return nil, listenErr
		}

		grpcServer = grpc.NewServer(
			grpc.MaxRecvMsgSize(4096),
			grpc.MaxSendMsgSize(65536),
		)

		pb.RegisterFileServiceServer(grpcServer, &rpcfiles.Service{
			NodeID:  c.ID,
			Catalog: cat,
		})

		grpcDone = make(chan struct{})

		go func() {
			defer close(grpcDone)

			if serveErr := grpcServer.Serve(ln); serveErr != nil && ctx.Err() == nil {
				log.Printf("gRPC stopped: %v", serveErr)
			}
		}()
	}

	var cleanupOnce sync.Once
	var cleanupErr error

	// Stop all application-owned services and release their resources.
	cleanup := func(shutdownCtx context.Context) error {
		cleanupOnce.Do(func() {
			var errs []error

			// Stop gRPC gracefully, forcing it to stop if the deadline expires.
			if grpcServer != nil {
				stopped := make(chan struct{})

				go func() {
					grpcServer.GracefulStop()
					close(stopped)
				}()

				select {
				case <-stopped:
				case <-shutdownCtx.Done():
					grpcServer.Stop()
					errs = append(errs, shutdownCtx.Err())
				}

				// Wait for the gRPC Serve goroutine within the same deadline.
				select {
				case <-grpcDone:
				case <-shutdownCtx.Done():
					errs = append(errs, shutdownCtx.Err())
				}
			}

			// Cancel IPC services, close listeners and wait for their workers.
			if err := stopIPC(shutdownCtx); err != nil {
				errs = append(errs, fmt.Errorf("IPC cleanup: %w", err))
			}

			// Stop peer forwarding before closing the event bus.
			cancelForward()
			if err := waitForWorkers(shutdownCtx, &forwardWG); err != nil {
				errs = append(errs, fmt.Errorf("forwarding cleanup: %w", err))
			}

			// Stop event consumers before closing the event bus.
			if err := stopConsumers(shutdownCtx); err != nil {
				errs = append(errs, fmt.Errorf("event consumer cleanup: %w", err))
			}

			if err := bus.Close(); err != nil {
				errs = append(errs, fmt.Errorf("event bus cleanup: %w", err))
			}

			cleanupErr = errors.Join(errs...)
		})

		return cleanupErr
	}

	return cleanup, nil
}

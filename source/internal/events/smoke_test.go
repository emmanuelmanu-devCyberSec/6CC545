package events

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// This checks supplied allocation only; it is not CW2-AT1 evidence.
func TestScaffoldAllocation(t *testing.T) {
	if _, err := New(0); err == nil {
		t.Fatal("zero capacity accepted")
	}
	b, err := New(2)
	if err != nil || b == nil {
		t.Fatalf("New: %v", err)
	}
}

// TestForwardRetriesFailedPeerSendThenDropsAndContinues verifies that
// Forward retries a failed peer send three times, drops that event,
// and continues forwarding subsequent events.
func TestForwardRetriesFailedPeerSendThenDropsAndContinues(t *testing.T) {
	b, err := New(4)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var failedAttempts atomic.Int32
	dropAttemptsDone := make(chan struct{})
	nextEventSent := make(chan struct{}, 1)

	// Simulate a peer transport that rejects the first event but
	// successfully accepts the next event.
	send := func(_ context.Context, event Event) error {
		if event.ID == "failed-event" {
			attempt := failedAttempts.Add(1)

			// Signal the test after all three attempts have occurred.
			if attempt == 3 {
				close(dropAttemptsDone)
			}

			return errors.New("peer send failed")
		}

		if event.ID == "next-event" {
			nextEventSent <- struct{}{}
			return nil
		}

		return errors.New("unexpected event")
	}

	// Start the forwarding worker.
	forwardDone := make(chan error, 1)
	go func() {
		forwardDone <- b.Forward(ctx, "file.added", send)
	}()

	// Publish an event once the forwarding subscription is active.
	// The bounded deadline prevents the test from waiting indefinitely.
	publishWhenReady := func(event Event) {
		t.Helper()

		deadline := time.NewTimer(2 * time.Second)
		defer deadline.Stop()

		for {
			delivery, publishErr := b.Publish(event)
			if publishErr != nil {
				t.Fatalf("Publish: %v", publishErr)
			}

			if delivery.Delivered > 0 {
				return
			}

			select {
			case <-deadline.C:
				t.Fatal("forwarding subscription did not become ready")
			default:
				runtime.Gosched()
			}
		}
	}

	publishWhenReady(Event{
		ID:     "failed-event",
		Kind:   "file.added",
		Origin: "node-a",
		Name:   "alpha.txt",
	})

	// Wait until the failed event has received all three attempts.
	select {
	case <-dropAttemptsDone:
	case <-time.After(2 * time.Second):
		t.Fatal("forwarding did not complete the three failed attempts")
	}

	publishWhenReady(Event{
		ID:     "next-event",
		Kind:   "file.added",
		Origin: "node-a",
		Name:   "diagram.txt",
	})

	// Confirm forwarding continued after dropping the failed event.
	select {
	case <-nextEventSent:
	case <-time.After(2 * time.Second):
		t.Fatal("forwarding did not continue after dropping the failed event")
	}

	if got := failedAttempts.Load(); got != 3 {
		t.Fatalf("failed event attempts = %d, want 3", got)
	}

	// Cancel the worker and wait for its shutdown.
	cancel()

	select {
	case err := <-forwardDone:
		if err != nil {
			t.Fatalf("Forward returned an error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("forwarding worker did not stop after cancellation")
	}
}

// TestConcurrentPublishWithSubscribersJoiningAndLeaving repeatedly publishes
// events while subscribers join, receive events, and leave the bus.
func TestConcurrentPublishWithSubscribersJoiningAndLeaving(t *testing.T) {
	b, err := New(4)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	// Coordinate the publisher and collect its delivery results.
	type publishResult struct {
		delivery Delivery
		err      error
	}

	publishRequests := make(chan Event)
	publishResults := make(chan publishResult)

	publisherCtx, stopPublisher := context.WithCancel(context.Background())
	defer stopPublisher()

	var publisherWG sync.WaitGroup
	publisherWG.Add(1)

	// Run one publisher worker for the duration of the integration test.
	go func() {
		defer publisherWG.Done()

		for {
			select {
			case <-publisherCtx.Done():
				return

			case event := <-publishRequests:
				delivery, publishErr := b.Publish(event)

				select {
				case publishResults <- publishResult{
					delivery: delivery,
					err:      publishErr,
				}:
				case <-publisherCtx.Done():
					return
				}
			}
		}
	}()

	const iterations = 50

	for i := 0; i < iterations; i++ {
		// A new subscriber joins for this publishing cycle.
		sub, err := b.Subscribe("file.added")
		if err != nil {
			t.Fatalf("iteration %d: Subscribe: %v", i, err)
		}

		// The subscriber signals readiness before publication begins.
		ready := make(chan struct{})
		received := make(chan Event, 1)
		subscriberDone := make(chan struct{})

		var subscriberWG sync.WaitGroup
		subscriberWG.Add(1)

		go func() {
			defer subscriberWG.Done()
			defer close(subscriberDone)

			close(ready)

			event, ok := <-sub.C
			if ok {
				received <- event
			}
		}()

		// Wait until the subscriber goroutine is ready.
		<-ready

		event := Event{
			ID:     fmt.Sprintf("concurrent-event-%d", i),
			Kind:   "file.added",
			Origin: "node-a",
			Name:   fmt.Sprintf("file-%d.txt", i),
		}

		// Ask the publisher worker to publish while this subscriber is active.
		select {
		case publishRequests <- event:
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: publisher was not ready", i)
		}

		// Confirm the event was delivered to the active subscriber.
		select {
		case result := <-publishResults:
			if result.err != nil {
				t.Fatalf("iteration %d: Publish: %v", i, result.err)
			}
			if result.delivery.Delivered != 1 {
				t.Fatalf(
					"iteration %d: delivered = %d, want 1",
					i, result.delivery.Delivered,
				)
			}
			if result.delivery.Dropped != 0 {
				t.Fatalf(
					"iteration %d: dropped = %d, want 0",
					i, result.delivery.Dropped,
				)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: publish result timed out", i)
		}

		select {
		case got := <-received:
			if got.ID != event.ID {
				t.Fatalf(
					"iteration %d: received event ID %q, want %q",
					i, got.ID, event.ID,
				)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: subscriber did not receive event", i)
		}

		// The subscriber leaves after receiving its event.
		sub.Cancel()

		// Wait for the subscriber goroutine to finish before the next cycle.
		select {
		case <-subscriberDone:
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: subscriber did not stop", i)
		}

		subscriberWG.Wait()
	}

	// Stop the publisher and wait for its goroutine to finish.
	stopPublisher()
	publisherWG.Wait()
}

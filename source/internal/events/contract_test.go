//go:build acceptance

package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func event(n string) Event {
	return Event{ID: n, Kind: "file.added", Origin: "node-a", Name: n + ".txt"}
}
func receive(t *testing.T, s *Subscription, want Event) {
	t.Helper()
	select {
	case got, ok := <-s.C:
		if !ok || got != want {
			t.Fatalf("event got=%+v open=%v want=%+v", got, ok, want)
		}
	case <-time.After(time.Second):
		t.Fatal("event delivery timed out")
	}
}
func TestTwoSubscribersAndTopics(t *testing.T) {
	b, _ := New(2)
	defer b.Close()
	a, e := b.Subscribe("file.added")
	if e != nil {
		t.Fatal(e)
	}
	c, e := b.Subscribe("file.added")
	if e != nil {
		t.Fatal(e)
	}
	other, e := b.Subscribe("peer.expired")
	if e != nil {
		t.Fatal(e)
	}
	d, e := b.Publish(event("one"))
	if e != nil || d != (Delivery{Delivered: 2}) {
		t.Fatalf("delivery %+v %v", d, e)
	}
	receive(t, a, event("one"))
	receive(t, c, event("one"))
	select {
	case <-other.C:
		t.Fatal("wrong topic delivered")
	default:
	}
}
func TestSlowSubscriberDropsNewest(t *testing.T) {
	b, _ := New(1)
	defer b.Close()
	s, e := b.Subscribe("file.added")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.Publish(event("first")); e != nil {
		t.Fatal(e)
	}
	result := make(chan Delivery, 1)
	failure := make(chan error, 1)
	go func() { d, e := b.Publish(event("second")); result <- d; failure <- e }()
	select {
	case d := <-result:
		if e := <-failure; e != nil || d != (Delivery{Dropped: 1}) {
			t.Fatalf("drop %+v %v", d, e)
		}
	case <-time.After(time.Second):
		t.Fatal("slow consumer blocked publisher")
	}
	receive(t, s, event("first"))
}
func TestCancelAndCloseAreIdempotent(t *testing.T) {
	b, _ := New(1)
	s, e := b.Subscribe("file.added")
	if e != nil {
		t.Fatal(e)
	}
	s.Cancel()
	s.Cancel()
	select {
	case _, ok := <-s.C:
		if ok {
			t.Fatal("cancel left channel open")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not close")
	}
	if e = b.Close(); e != nil {
		t.Fatal(e)
	}
	if e = b.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = b.Publish(event("late")); !errors.Is(e, ErrClosed) {
		t.Fatalf("publish after close: %v", e)
	}
	if _, e = b.Subscribe("file.added"); !errors.Is(e, ErrClosed) {
		t.Fatalf("subscribe after close: %v", e)
	}
}
func TestPublishCancelCloseConcurrent(t *testing.T) {
	b, _ := New(8)
	s, e := b.Subscribe("file.added")
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = b.Publish(event("concurrent")); s.Cancel(); _ = b.Close() }()
	}
	wg.Wait()
}
func TestInvalidEventRejected(t *testing.T) {
	b, _ := New(1)
	defer b.Close()
	if _, e := b.Subscribe(""); e == nil {
		t.Fatal("empty topic accepted")
	}
	if _, e := b.Publish(Event{}); e == nil {
		t.Fatal("empty event accepted")
	}
}

func TestForwardDeliversThenCancels(t *testing.T) {
	b, _ := New(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sent := make(chan Event, 1)
	done := make(chan error, 1)
	go func() {
		done <- b.Forward(ctx, "file.added", func(ctx context.Context, e Event) error {
			select {
			case sent <- e:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	// Publish until the worker has subscribed; this is bounded and does not require
	// an implementation-specific sleep to assume goroutine scheduling order.
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			_, _ = b.Publish(event("forward"))
		case got := <-sent:
			if got != event("forward") {
				t.Fatal("forward changed event")
			}
			cancel()
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				t.Fatal("forward ignored cancellation")
			}
		case e := <-done:
			t.Fatalf("forward stopped early: %v", e)
		case <-deadline.C:
			t.Fatal("forward did not deliver")
		}
	}
}

// Package events is an assessed process-local publish/subscribe wireframe.
package events

import (
	"errors"
	"sync"
)

var ErrNotImplemented = errors.New("TODO W06: event bus not implemented")
var ErrClosed = errors.New("event bus closed")

// Kind is the topic. ID is unique within the origin node's event history.
// Only publish metadata; never put file bodies, tokens or keys in an event.
type Event struct{ ID, Kind, Origin, Name string }
type Delivery struct{ Delivered, Dropped int }
type subscriber struct {
	topic string
	ch    chan Event
}
type Bus struct {
	mu          sync.Mutex
	capacity    int
	next        uint64
	closed      bool
	subscribers map[uint64]subscriber
}
type Subscription struct {
	C      <-chan Event
	cancel func()
}

func (s *Subscription) Cancel() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

// New is supplied allocation infrastructure. A bounded queue is mandatory.
func New(capacity int) (*Bus, error) {
	if capacity < 1 || capacity > 1024 {
		return nil, errors.New("capacity must be 1..1024")
	}
	return &Bus{capacity: capacity, subscribers: make(map[uint64]subscriber)}, nil
}

func (b *Bus) Subscribe(topic string) (*Subscription, error) {
	// TODO 6.2: reject empty/oversize topics and closed bus; allocate an owned
	// bounded channel and a unique subscription ID. Cancel must be idempotent.
	// Bus owns channel closure. Never expose a writable channel.

	// A subscription must have a topic so the bus knows which events to deliver.
	if topic == "" {
		return nil, errors.New("topic must not be empty")
	}

	// Lock the bus because its state and subscriber map may be accessed
	// by multiple goroutines at the same time.
	b.mu.Lock()
	defer b.mu.Unlock()

	// Do not allow new subscriptions after the bus has been closed.
	if b.closed {
		return nil, ErrClosed
	}

	// Generate a unique ID for this subscription.
	b.next++
	id := b.next

	// Create a buffered channel using the capacity configured when the bus
	// was created. This gives each subscriber an independent bounded queue.
	ch := make(chan Event, b.capacity)

	// Register the subscriber with its topic and event channel.
	b.subscribers[id] = subscriber{
		topic: topic,
		ch:    ch,
	}

	// Return a subscription with a receive-only channel so the subscriber
	// can receive events but cannot send events into the bus.
	return &Subscription{
		C: ch,
		cancel: func() {
			// Lock the bus before changing the subscriber map or closing
			// the channel. This prevents conflicts with Publish and Close.
			b.mu.Lock()
			defer b.mu.Unlock()

			// Check that the subscription is still active. This makes
			// repeated calls to Cancel safe.
			if sub, ok := b.subscribers[id]; ok {
				// Remove the subscriber before closing its channel.
				delete(b.subscribers, id)
				close(sub.ch)
			}
		},
	}, nil
}

func (b *Bus) Publish(e Event) (Delivery, error) {
	// TODO 6.3: validate e. Deliver a copy to each matching active subscriber.
	// Use nonblocking queue sends with drop-newest on a full queue. Return
	// exact Delivered/Dropped counts. Coordinate sends and closure with mu.
	// Do not spawn an unbounded goroutine per subscriber or block on a reader.

	// Reject events that are missing any required metadata field.
	if e.ID == "" || e.Kind == "" || e.Origin == "" || e.Name == "" {
		return Delivery{}, errors.New("event fields must not be empty")
	}

	// Protect the subscriber map and coordinate publishing with cancellation
	// and bus closure.
	b.mu.Lock()
	defer b.mu.Unlock()

	// Reject publishing if the bus has already been closed.
	if b.closed {
		return Delivery{}, ErrClosed
	}

	// Keep track of successful deliveries and events dropped because
	// a subscriber's queue is full.
	var result Delivery

	// Visit each registered subscriber independently.
	for _, sub := range b.subscribers {

		// Skip subscribers that are listening to a different topic.
		if sub.topic != e.Kind {
			continue
		}

		// Attempt to add the event without waiting for the subscriber.
		select {
		case sub.ch <- e:
			// The event was accepted into this subscriber's queue.
			result.Delivered++

		default:
			// The queue is full. Drop the newest event instead of
			// blocking the publisher or removing an older queued event.
			result.Dropped++
		}
	}

	// Return the actual delivery and drop counts.
	return result, nil
}

func (b *Bus) Close() error {
	// TODO 6.4: atomically mark closed, close each queue once, empty the map.
	// Repeated Close returns nil. A later Publish/Subscribe returns ErrClosed.

	// Lock the bus to prevent concurrent publishing, subscribing
	// or cancellation while the queues are being closed.
	b.mu.Lock()
	defer b.mu.Unlock()

	// If the bus is already closed, return successfully.
	// This makes repeated calls to Close safe.
	if b.closed {
		return nil
	}

	// Mark the bus as closed so no new subscriptions or publications
	// can be accepted.
	b.closed = true

	// Close every active subscriber channel.
	// The bus owns these channels, so it is responsible for closing them.
	for id, sub := range b.subscribers {
		close(sub.ch)

		// Remove the subscriber after closing its channel.
		delete(b.subscribers, id)
	}

	// All subscriber queues have been closed and removed.
	return nil
}

package events

import (
	"context"
	"time"
)

// Forward is an assessed adapter loop. The send callback uses the student's
// current transport; in the final system it MUST authenticate/encrypt peer traffic.
func (b *Bus) Forward(ctx context.Context, topic string, send func(context.Context, Event) error) error {
	// Subscribe once to the requested topic.
	sub, err := b.Subscribe(topic)
	if err != nil {
		return err
	}

	// Release the subscription when forwarding stops.
	defer sub.Cancel()

	// Limit each transport send so a blocked peer cannot hold the worker forever.
	const sendTimeout = time.Second

	// Allow one initial attempt and two retries for a failed send.
	const maxAttempts = 3

	for {
		select {
		// Stop forwarding when the caller cancels the context.
		case <-ctx.Done():
			return nil

		// Receive the next event from the subscription.
		case event, ok := <-sub.C:
			// Stop if the subscription channel has been closed.
			if !ok {
				return nil
			}

			// Attempt to send the event using the bounded retry policy.
			for attempt := 0; attempt < maxAttempts; attempt++ {
				// Give this individual send a finite deadline.
				sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
				err := send(sendCtx, event)
				cancel()

				// A successful send needs no further attempts.
				if err == nil {
					break
				}

				// Stop retrying if forwarding has been cancelled.
				if ctx.Err() != nil {
					return nil
				}

				// Drop the event after the third failed attempt.
				// Forwarding then continues with the next event.
				if attempt == maxAttempts-1 {
					break
				}
			}
		}
	}
}

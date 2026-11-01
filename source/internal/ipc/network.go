package ipc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type OnMessage func(Message)

func deadline(ctx context.Context, timeout time.Duration) time.Time {
	d := time.Now().Add(timeout)
	if cd, ok := ctx.Deadline(); ok && cd.Before(d) {
		return cd
	}
	return d
}
func makeAck(id string, m Message) Ack {
	return Ack{Version: Version, RequestID: m.RequestID, Receiver: id, Accepted: true}
}
func checkAck(a Ack, m Message) error {
	if a.Version != Version || a.RequestID != m.RequestID || !a.Accepted || a.Receiver == "" {
		return errors.New("invalid or unrelated acknowledgement")
	}
	return nil
}
func ServeTCP(ctx context.Context, ln net.Listener, id string, timeout time.Duration, on OnMessage) error {
	// Close the listener when cancellation allows the server to stop accepting.
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	var handlers sync.WaitGroup
	sem := make(chan struct{}, 32)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Wait for active handlers before returning from shutdown.
			handlers.Wait()
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		// Give each connection a bounded lifetime and limit active handlers.
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			_ = conn.Close()
			handlers.Wait()
			return nil
		}

		handlers.Add(1)
		go func(conn net.Conn) {
			defer handlers.Done()
			defer func() { <-sem }()
			defer conn.Close()

			// Bound this connection's work by the timeout and server context.
			op, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			if err := conn.SetDeadline(deadline(op, timeout)); err != nil {
				return
			}

			// Read and validate one message before invoking the callback.
			data, err := ReadFrame(conn)
			if err != nil {
				return
			}
			var m Message
			if err := Decode(data, &m); err != nil {
				return
			}
			if err := m.Validate(); err != nil {
				return
			}

			if on != nil {
				on(m)
			}

			// Acknowledge the request using its matching request ID.
			ackData, err := Encode(makeAck(id, m))
			if err != nil {
				return
			}
			_ = WriteFrame(conn, ackData)
		}(conn)
	}
}
func SendTCP(ctx context.Context, address string, m Message, timeout time.Duration) (Ack, error) {
	// Validate the outgoing message before opening a connection.
	if err := m.Validate(); err != nil {
		return Ack{}, err
	}

	// Derive a context with the operation timeout.
	op, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Connect using the operation context.
	conn, err := (&net.Dialer{}).DialContext(op, "tcp", address)
	if err != nil {
		return Ack{}, err
	}
	defer conn.Close()

	// Apply a deadline that respects both the timeout and context.
	if err := conn.SetDeadline(deadline(op, timeout)); err != nil {
		return Ack{}, err
	}

	// Encode and send one framed message.
	data, err := Encode(m)
	if err != nil {
		return Ack{}, err
	}
	if err := WriteFrame(conn, data); err != nil {
		return Ack{}, err
	}

	// Read and decode one framed acknowledgement.
	data, err = ReadFrame(conn)
	if err != nil {
		return Ack{}, err
	}
	var ack Ack
	if err := Decode(data, &ack); err != nil {
		return Ack{}, err
	}

	// Reject acknowledgements that do not match this request.
	if err := checkAck(ack, m); err != nil {
		return Ack{}, err
	}

	return ack, nil
}
func ServeUDP(ctx context.Context, c *net.UDPConn, id string, timeout time.Duration, on OnMessage) error {
	// Close the socket when the context is cancelled.
	go func() {
		<-ctx.Done()
		_ = c.Close()
	}()

	// Allow one extra byte so oversized datagrams can be rejected.
	buf := make([]byte, MaxMessage+1)

	for {
		if err := c.SetReadDeadline(deadline(ctx, timeout)); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		n, addr, err := c.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}

			return err
		}

		// Reject oversized datagrams and malformed messages.
		if n == 0 || n > MaxMessage {
			continue
		}

		var m Message
		if err := Decode(buf[:n], &m); err != nil {
			continue
		}
		if err := m.Validate(); err != nil {
			continue
		}

		// Record only valid messages.
		if on != nil {
			on(m)
		}

		// Send the acknowledgement directly to the sender.
		data, err := Encode(makeAck(id, m))
		if err != nil {
			continue
		}

		if _, err := c.WriteToUDP(data, addr); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}
func SendUDP(ctx context.Context, address string, m Message, timeout time.Duration) (Ack, error) {
	// Validate the outgoing message before sending.
	if err := m.Validate(); err != nil {
		return Ack{}, err
	}

	// Bound the complete request and response operation.
	op, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Resolve and connect the UDP endpoint.
	remote, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return Ack{}, err
	}
	conn, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return Ack{}, err
	}
	defer conn.Close()

	// Set the deadline to respect the operation's remaining time.
	if err := conn.SetDeadline(deadline(op, timeout)); err != nil {
		return Ack{}, err
	}

	// Encode the message as one UDP datagram, without TCP framing.
	data, err := Encode(m)
	if err != nil {
		return Ack{}, err
	}
	if _, err := conn.Write(data); err != nil {
		return Ack{}, err
	}

	// Wait for one acknowledgement within the same operation budget.
	buf := make([]byte, MaxMessage)
	n, err := conn.Read(buf)
	if err != nil {
		return Ack{}, err
	}

	// Decode and validate the acknowledgement's correlation fields.
	var ack Ack
	if err := Decode(buf[:n], &ack); err != nil {
		return Ack{}, err
	}
	if err := checkAck(ack, m); err != nil {
		return Ack{}, err
	}

	return ack, nil
}

var _ = fmt.Errorf
var _ = log.Printf
var _ sync.WaitGroup
var _ = deadline
var _ = makeAck
var _ = checkAck

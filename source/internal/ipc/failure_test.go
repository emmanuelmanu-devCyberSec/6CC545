package ipc

import (
	"context"
	"encoding/binary"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestWeek04_ServerRejectsOversizeTCP(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer ln.Close()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() { done <- ServeTCP(ctx, ln, "node-b", 100*time.Millisecond, func(Message) { calls.Add(1) }) }()
	c, e := net.DialTimeout("tcp", ln.Addr().String(), 200*time.Millisecond)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(300 * time.Millisecond))
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], MaxMessage+1)
	_, _ = c.Write(h[:])
	var b [1]byte
	n, e := c.Read(b[:])
	if e == nil || n != 0 {
		t.Error("oversize TCP frame was not rejected")
	}
	if calls.Load() != 0 {
		t.Error("rejected frame mutated inbox")
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Errorf("serve failed: %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown hung")
	}
}
func TestWeek04_UDPDropAndDeadline(t *testing.T) {
	c, e := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer c.Close()
	done := make(chan error, 1)
	var calls atomic.Int32
	go func() { done <- ServeUDP(ctx, c, "node-b", 100*time.Millisecond, func(Message) { calls.Add(1) }) }()
	remote, e := net.DialTimeout("udp", c.LocalAddr().String(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	defer remote.Close()
	_ = remote.SetDeadline(time.Now().Add(150 * time.Millisecond))
	_, _ = remote.Write([]byte(`{malformed}`))
	var b [128]byte
	n, e := remote.Read(b[:])
	if e == nil || n != 0 {
		t.Error("malformed UDP request should be dropped")
	}
	if calls.Load() != 0 {
		t.Error("rejected datagram changed inbox")
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Errorf("serve: %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("UDP shutdown hung")
	}
}

func TestWeek04_UDPRejectsOversizeDatagram(t *testing.T) {
	// Start a UDP listener on an available loopback port.
	c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Create a context to control the server's lifetime.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Track whether the server incorrectly records a rejected message.
	var calls atomic.Int32
	done := make(chan error, 1)

	// Run the UDP server in the background.
	go func() {
		done <- ServeUDP(ctx, c, "node-b", 100*time.Millisecond, func(Message) {
			calls.Add(1)
		})
	}()

	// Connect a UDP client to the server's listening address.
	remote, err := net.DialTimeout("udp", c.LocalAddr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()

	// Send a datagram one byte larger than the allowed message size.
	if _, err := remote.Write(make([]byte, MaxMessage+1)); err != nil {
		t.Fatal(err)
	}

	// Set a deadline so the test cannot wait indefinitely for a response.
	_ = remote.SetReadDeadline(time.Now().Add(150 * time.Millisecond))

	// A rejected oversized datagram should not receive an acknowledgement.
	var b [128]byte
	n, err := remote.Read(b[:])
	if err == nil || n != 0 {
		t.Error("oversized UDP datagram should be dropped without a response")
	}

	// Confirm the rejected datagram did not trigger the inbox callback.
	if calls.Load() != 0 {
		t.Error("oversized UDP datagram triggered the inbox callback")
	}

	// Cancel the server and verify that it shuts down promptly.
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("UDP shutdown hung")
	}
}

func TestWeek04_TCPIncompleteFrameTimeout(t *testing.T) {
	// Start a TCP listener on an available loopback port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Create a context to control the server's lifetime.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Track whether an incomplete frame reaches the inbox callback.
	var calls atomic.Int32
	done := make(chan error, 1)

	// Run the TCP server with a short per-connection timeout.
	go func() {
		done <- ServeTCP(ctx, ln, "node-b", 150*time.Millisecond, func(Message) {
			calls.Add(1)
		})
	}()

	// Connect to the server.
	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Send only part of the four-byte frame header.
	if _, err := conn.Write([]byte{0, 0}); err != nil {
		t.Fatal(err)
	}

	// Wait for the server to close the incomplete connection.
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	_, err = conn.Read(b[:])
	if err == nil {
		t.Fatal("incomplete TCP frame connection was not closed")
	}

	// Confirm the incomplete frame did not reach the inbox callback.
	if calls.Load() != 0 {
		t.Error("incomplete TCP frame triggered the inbox callback")
	}

	// Cancel the server and verify prompt shutdown.
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("TCP server shutdown hung")
	}
}

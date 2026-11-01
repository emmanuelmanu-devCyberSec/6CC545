package ipc

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWeek04_RoundTrips(t *testing.T) {
	for _, network := range []string{"tcp", "udp"} {
		t.Run(network, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			var address string
			if network == "tcp" {
				ln, e := net.Listen("tcp", "127.0.0.1:0")
				if e != nil {
					t.Fatal(e)
				}
				defer ln.Close()
				address = ln.Addr().String()
				go func() { done <- ServeTCP(ctx, ln, "node-b", 200*time.Millisecond, nil) }()
			} else {
				c, e := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
				if e != nil {
					t.Fatal(e)
				}
				defer c.Close()
				address = c.LocalAddr().String()
				go func() { done <- ServeUDP(ctx, c, "node-b", 200*time.Millisecond, nil) }()
			}
			m := Message{1, "r1", "node-a", "hello"}
			op, stop := context.WithTimeout(ctx, 500*time.Millisecond)
			defer stop()
			var a Ack
			var e error
			if network == "tcp" {
				a, e = SendTCP(op, address, m, 300*time.Millisecond)
			} else {
				a, e = SendUDP(op, address, m, 300*time.Millisecond)
			}
			if e != nil || a.Receiver != "node-b" || a.RequestID != "r1" {
				t.Errorf("round trip failed: %+v %v", a, e)
			}
			cancel()
			select {
			case e := <-done:
				if e != nil {
					t.Errorf("shutdown: %v", e)
				}
			case <-time.After(time.Second):
				t.Fatal("server did not stop")
			}
		})
	}
}
func TestWeek04_CancelledClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := Message{1, "r1", "node-a", "hello"}
	for _, n := range []string{"tcp", "udp"} {
		var e error
		if n == "tcp" {
			_, e = SendTCP(ctx, "127.0.0.1:9", m, 100*time.Millisecond)
		} else {
			_, e = SendUDP(ctx, "127.0.0.1:9", m, 100*time.Millisecond)
		}
		if e == nil {
			t.Errorf("%s ignored cancelled context", n)
		}
	}
}

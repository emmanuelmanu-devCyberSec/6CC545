package app

import (
	"context"
	"example.edu/6cc545/cw1/internal/config"
	"net"
	"strings"
	"testing"
	"time"
)

func TestWeek01_OccupiedPort(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	c := config.Config{ID: "node-a", HTTP: ln.Addr().String(), Timeout: time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	e = Run(ctx, c)
	if e == nil || !strings.Contains(e.Error(), "listen HTTP") {
		t.Fatalf("want clear bind error, got %v", e)
	}
}
func TestWeek01_CancelledLifecycle(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := ln.Addr().String()
	_ = ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := config.Config{ID: "node-a", HTTP: address, Timeout: time.Second}
	if e = Run(ctx, c); e != nil {
		t.Fatal(e)
	}
}

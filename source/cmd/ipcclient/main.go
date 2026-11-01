package main

import (
	"context"
	"encoding/json"
	"example.edu/6cc545/cw1/internal/ipc"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	network := flag.String("network", "tcp", "tcp or udp")
	target := flag.String("target", "127.0.0.1:7002", "target socket")
	id := flag.String("id", "node-a", "sender ID")
	text := flag.String("text", "hello", "message text")
	request := flag.String("request", "demo-1", "request ID")
	timeout := flag.Duration("timeout", 2*time.Second, "overall timeout")
	flag.Parse()
	if *timeout <= 0 || *timeout > 30*time.Second {
		fmt.Fprintln(os.Stderr, "timeout must be >0 and <=30s")
		os.Exit(2)
	}
	ctx, stop := context.WithTimeout(context.Background(), *timeout)
	defer stop()
	m := ipc.Message{Version: 1, RequestID: *request, NodeID: *id, Text: *text}
	var a ipc.Ack
	var e error
	switch *network {
	case "tcp":
		a, e = ipc.SendTCP(ctx, *target, m, *timeout)
	case "udp":
		a, e = ipc.SendUDP(ctx, *target, m, *timeout)
	default:
		e = fmt.Errorf("network must be tcp or udp")
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(a)
}

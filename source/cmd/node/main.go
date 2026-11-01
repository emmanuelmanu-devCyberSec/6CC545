package main

import (
	"context"
	"errors"
	"example.edu/6cc545/cw1/internal/app"
	"example.edu/6cc545/cw1/internal/config"
	"flag"
	"log"
	"os"
	"os/signal"
)

func main() {
	c, e := config.Parse(os.Args[1:], os.Stderr)
	if errors.Is(e, flag.ErrHelp) {
		return
	}
	if e != nil {
		log.Printf("configuration: %v", e)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if e = app.Run(ctx, c); e != nil {
		log.Printf("node stopped: %v", e)
		os.Exit(1)
	}
}

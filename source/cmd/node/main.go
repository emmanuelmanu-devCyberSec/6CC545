package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"example.edu/6cc545/cw1/internal/app"
	"example.edu/6cc545/cw1/internal/config"
)

func main() {
	// Parse configuration before starting the application.
	c, err := config.Parse(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		log.Printf("configuration: %v", err)
		os.Exit(2)
	}

	// Handle both terminal interrupts and Docker termination signals.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Run the node until it stops or receives a termination signal.
	if err := app.Run(ctx, c); err != nil {
		log.Printf("node stopped: %v", err)
		os.Exit(1)
	}
}

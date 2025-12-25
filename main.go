package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/henryaj/autoclaude/internal/watcher"
)

var version = "1.0.0"

func main() {
	var (
		verbose     bool
		showVersion bool
		testMode    bool
	)

	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&testMode, "test", false, "Test mode: wait 10s then send resume sequence")
	flag.Parse()

	if showVersion {
		fmt.Printf("autoclaude v%s\n", version)
		os.Exit(0)
	}

	w, err := watcher.New(verbose, testMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure you're running this inside a tmux session.")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	if err := w.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

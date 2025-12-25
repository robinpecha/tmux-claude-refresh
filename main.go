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

var version = "dev"

const usage = `autoclaude - Automatically resume Claude Code sessions after rate limits

USAGE:
    autoclaude [OPTIONS]

DESCRIPTION:
    autoclaude monitors other tmux panes in the current window for Claude Code
    rate limit messages. When a limit is detected, it waits for the reset time,
    then automatically sends the resume command to continue the session.

    Run this in a separate tmux pane alongside your Claude Code sessions.
    It will monitor all other panes in the same window.

OPTIONS:
    -v          Enable verbose/debug logging
    -version    Show version information
    -test       Test mode: wait 10s then send resume sequence to another pane

EXAMPLE:
    # Split your tmux window and run autoclaude in one pane
    tmux split-window -h
    autoclaude

    # With verbose logging
    autoclaude -v

HOW IT WORKS:
    1. Polls all tmux panes in the current window every 5 seconds
    2. Detects rate limit messages (e.g., "Usage limit reached")
    3. Parses the reset time from the message
    4. Waits until the limit resets, plus a random 5-10 second delay
    5. Sends Enter (to dismiss any selector menu) then "continue" + Enter

REQUIREMENTS:
    - Must be run inside a tmux session
    - Claude Code sessions must be in other panes of the same window
`

func main() {
	var (
		verbose     bool
		showVersion bool
		testMode    bool
	)

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

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

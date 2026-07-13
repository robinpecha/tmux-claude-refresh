package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robinpecha/tmux-claude-refresh/internal/config"
	"github.com/robinpecha/tmux-claude-refresh/internal/tmux"
	"github.com/robinpecha/tmux-claude-refresh/internal/tui"
)

var version = "dev"

func main() {
	testPattern := flag.String("test-pattern", "", "Test mode: trigger auto-continue when this string is found (for debugging)")
	flag.Parse()

	// Validate tmux environment
	if err := tmux.CheckTmuxEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Load()

	p := tea.NewProgram(
		tui.New(version, *testPattern, cfg.DisplayLocation),
		tea.WithAltScreen(),
	)

	// Handle SIGINT and SIGTERM to ensure clean exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robinpecha/tmux-claude-refresh/internal/config"
	"github.com/robinpecha/tmux-claude-refresh/internal/tmux"
	"github.com/robinpecha/tmux-claude-refresh/internal/tui"
)

var version = "dev"

const helpText = `tmux-claude-refresh v%s

A TUI that watches tmux panes running Claude Code and automatically
sends "continue" when a rate limit resets.

Usage:
  tmux-claude-refresh [flags]

Flags:
  -h, --help        Show this help
  -v, --version     Print version and exit
  -u, --update      Print the update command and exit
  --test-pattern    Test mode: trigger auto-continue when this string is found

Keys (inside the TUI):
  ←↑↓→              Navigate between panes
  tab               Toggle auto-continue for selected pane
  a                 Enable auto-continue for all Claude Code panes
  n                 Disable auto-continue for all Claude Code panes
  r                 Refresh pane layout
  t                 Choose display timezone
  b                 Toggle terminal bell on auto-continue
  h / ?             Show help
  q                 Quit

Pane colors:
  Orange            Claude Code (auto-continue off)
  Green             Claude Code (auto-continue on)
  Red               Rate limited (waiting for reset)
  Cyan              Selected pane

Run it inside a tmux session. Requires tmux.
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), helpText, version)
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "Print version and exit")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")

	var showUpdate bool
	flag.BoolVar(&showUpdate, "u", false, "Print the update command and exit")
	flag.BoolVar(&showUpdate, "update", false, "Print the update command and exit")

	testPattern := flag.String("test-pattern", "", "Test mode: trigger auto-continue when this string is found (for debugging)")
	flag.Parse()

	if showVersion {
		fmt.Printf("tmux-claude-refresh v%s\n", version)
		return
	}

	if showUpdate {
		printUpdateCommand()
		return
	}

	// Validate tmux environment
	if err := tmux.CheckTmuxEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := config.Load()

	p := tea.NewProgram(
		tui.New(version, *testPattern, cfg.DisplayLocation, cfg.BellEnabled),
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

// printUpdateCommand detects the running binary's directory and OS/arch,
// then prints the appropriate curl|tar command to update in place.
func printUpdateCommand() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine binary path: %v\n", err)
		os.Exit(1)
	}
	dir := filepath.Dir(exe)

	// Build the download URL for the user's platform.
	url := fmt.Sprintf(
		"https://github.com/robinpecha/tmux-claude-refresh/releases/latest/download/tmux-claude-refresh_%s_%s.tar.gz",
		runtime.GOOS, runtime.GOARCH,
	)

	// Omit sudo if the target directory is writable by the current user.
	sudo := "sudo "
	if f, err := os.Stat(dir); err == nil && f.IsDir() {
		if tmp, err := os.CreateTemp(dir, ".tcr-write-test-*"); err == nil {
			tmp.Close()
			os.Remove(tmp.Name())
			sudo = ""
		}
	}

	fmt.Printf("To update this tool to latest release, paste this command:\n")
	fmt.Printf("curl -sL %s | %star -xz -C %s tmux-claude-refresh\n", url, sudo, dir)
}

package watcher

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/henryaj/autoclaude/internal/detector"
	"github.com/henryaj/autoclaude/internal/tmux"
)

const (
	pollInterval = 5 * time.Second
	minDelay     = 5 * time.Second
	maxDelay     = 10 * time.Second
	resumeMsg    = "continue"
)

// Watcher monitors tmux panes for Claude usage limits.
type Watcher struct {
	window   string
	verbose  bool
	testMode bool
}

// New creates a new Watcher for the current tmux window.
func New(verbose bool, testMode bool) (*Watcher, error) {
	if err := tmux.ValidateEnvironment(); err != nil {
		return nil, err
	}

	window, err := tmux.GetCurrentWindow()
	if err != nil {
		return nil, fmt.Errorf("failed to get current window: %w", err)
	}

	return &Watcher{
		window:   window,
		verbose:  verbose,
		testMode: testMode,
	}, nil
}

// Run starts the main polling loop.
func (w *Watcher) Run(ctx context.Context) error {
	if w.testMode {
		return w.runTestMode(ctx)
	}

	w.log("Starting autoclaude watcher...")
	w.log("Watching tmux window: %s", w.window)
	w.log("Poll interval: %v", pollInterval)
	w.log("Resume delay: %v-%v", minDelay, maxDelay)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Do an initial poll immediately
	w.pollOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log("Shutting down...")
			return nil
		case <-ticker.C:
			w.pollOnce(ctx)
		}
	}
}

const testModeDelay = 10 * time.Second

// runTestMode simulates the resume sequence after a 10 second delay.
func (w *Watcher) runTestMode(ctx context.Context) error {
	w.log("TEST MODE: Will send resume sequence in %v", testModeDelay)

	panes, err := tmux.ListPanes(w.window)
	if err != nil {
		return fmt.Errorf("failed to list panes: %w", err)
	}

	currentPane, _ := tmux.GetCurrentPane()

	// Find the first pane that isn't the current one
	var targetPane string
	for _, pane := range panes {
		if pane != currentPane {
			targetPane = pane
			break
		}
	}

	if targetPane == "" {
		return fmt.Errorf("no other panes found in window %s", w.window)
	}

	w.log("TEST MODE: Target pane: %s", targetPane)
	w.log("TEST MODE: Waiting %v...", testModeDelay)

	select {
	case <-ctx.Done():
		return nil
	case <-time.After(testModeDelay):
	}

	w.log("TEST MODE: Sending Enter to dismiss selector menu")
	if err := tmux.SendEnter(targetPane); err != nil {
		return fmt.Errorf("failed to send Enter: %w", err)
	}

	time.Sleep(500 * time.Millisecond)

	w.log("TEST MODE: Sending resume command: %q", resumeMsg)
	if err := tmux.SendKeys(targetPane, resumeMsg); err != nil {
		return fmt.Errorf("failed to send resume: %w", err)
	}

	w.log("TEST MODE: Complete")
	return nil
}

func (w *Watcher) pollOnce(ctx context.Context) {
	w.debug("--- Poll cycle starting ---")

	panes, err := tmux.ListPanes(w.window)
	if err != nil {
		w.log("ERROR: Failed to list panes: %v", err)
		return
	}

	currentPane, _ := tmux.GetCurrentPane()
	w.debug("Found %d pane(s) in window %s: %v", len(panes), w.window, panes)
	w.debug("Current pane (will skip): %s", currentPane)

	for _, pane := range panes {
		// Skip the pane we're running in
		if pane == currentPane {
			w.debug("Skipping pane %s (current pane)", pane)
			continue
		}

		w.debug("Checking pane %s...", pane)

		content, err := tmux.CapturePaneContent(pane)
		if err != nil {
			w.log("ERROR: Failed to capture pane %s: %v", pane, err)
			continue
		}

		// Show last few lines of content in debug mode
		w.debug("Pane %s content (last 5 lines):\n%s", pane, lastNLines(content, 5))

		limitInfo := detector.DetectUsageLimit(content)
		if limitInfo.Detected {
			w.debug("Limit DETECTED in pane %s", pane)
			w.handleLimit(ctx, pane, limitInfo)
		} else {
			w.debug("No limit detected in pane %s", pane)
		}
	}

	w.debug("--- Poll cycle complete ---")
}

// lastNLines returns the last n non-empty lines of a string.
func lastNLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 0 {
		return "(empty)"
	}

	// Filter out empty lines from the end
	var nonEmpty []string
	for i := len(lines) - 1; i >= 0 && len(nonEmpty) < n; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			nonEmpty = append([]string{line}, nonEmpty...)
		}
	}

	if len(nonEmpty) == 0 {
		return "(empty)"
	}

	// Indent for readability
	var result []string
	for _, line := range nonEmpty {
		// Truncate long lines
		if len(line) > 80 {
			line = line[:77] + "..."
		}
		result = append(result, "    "+line)
	}
	return strings.Join(result, "\n")
}

func (w *Watcher) handleLimit(ctx context.Context, pane string, limitInfo *detector.LimitInfo) {
	w.log("Usage limit detected in pane %s: %s", pane, limitInfo.RawMessage)
	w.log("Format: %s", limitInfo.Format)
	w.log("Reset time: %v", limitInfo.ResetTime.Format("2006-01-02 15:04:05"))

	// Wait until reset time
	waitDuration := time.Until(limitInfo.ResetTime)
	if waitDuration > 0 {
		w.log("Waiting %v until limit resets...", waitDuration.Round(time.Second))

		select {
		case <-ctx.Done():
			return
		case <-time.After(waitDuration):
		}
	}

	// Add random delay between 5-10 seconds
	delay := minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay)))
	w.log("Limit lifted, waiting %v before resuming...", delay.Round(time.Second))

	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}

	// First, send Enter to dismiss any selector menu (e.g., rate limit options)
	w.log("Sending Enter to dismiss selector menu in pane %s", pane)
	if err := tmux.SendEnter(pane); err != nil {
		w.log("ERROR: Failed to send Enter: %v", err)
		return
	}

	// Brief pause to let the UI respond
	time.Sleep(500 * time.Millisecond)

	// Send resume command: "continue" + Enter
	w.log("Sending resume command to pane %s: %q", pane, resumeMsg)
	if err := tmux.SendKeys(pane, resumeMsg); err != nil {
		w.log("ERROR: Failed to send resume: %v", err)
		return
	}

	w.log("Resume command sent successfully")
}

func (w *Watcher) log(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [INFO] %s\n", timestamp, msg)
}

func (w *Watcher) debug(format string, args ...interface{}) {
	if w.verbose {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("[%s] [DEBUG] %s\n", timestamp, msg)
	}
}

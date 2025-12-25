package tmux

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrNotInTmux     = errors.New("not running inside a tmux session")
	ErrNoPaneEnv     = errors.New("TMUX_PANE environment variable not set")
	ErrCommandFailed = errors.New("tmux command failed")
)

// ValidateEnvironment checks that we're running inside a tmux session.
func ValidateEnvironment() error {
	if os.Getenv("TMUX") == "" {
		return ErrNotInTmux
	}
	return nil
}

// GetCurrentPane returns the current pane ID from the TMUX_PANE environment variable.
func GetCurrentPane() (string, error) {
	pane := os.Getenv("TMUX_PANE")
	if pane == "" {
		return "", ErrNoPaneEnv
	}
	return pane, nil
}

// GetCurrentWindow returns the window ID for the current pane.
func GetCurrentWindow() (string, error) {
	pane, err := GetCurrentPane()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("tmux", "display-message", "-p", "-t", pane, "#{window_id}")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Join(ErrCommandFailed, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ListPanes returns all pane IDs in the specified window.
func ListPanes(window string) ([]string, error) {
	cmd := exec.Command("tmux", "list-panes", "-t", window, "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Join(ErrCommandFailed, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var panes []string
	for _, line := range lines {
		if line != "" {
			panes = append(panes, line)
		}
	}

	return panes, nil
}

// CapturePaneContent captures the visible content of a pane.
func CapturePaneContent(pane string) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-p", "-t", pane)
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Join(ErrCommandFailed, err)
	}

	return string(output), nil
}

// SendKeys sends text to a pane followed by Enter.
func SendKeys(pane string, text string) error {
	// Send text first
	cmd := exec.Command("tmux", "send-keys", "-t", pane, text)
	if err := cmd.Run(); err != nil {
		return errors.Join(ErrCommandFailed, err)
	}

	// Send Enter as a separate command
	return SendEnter(pane)
}

// SendEnter sends an Enter keypress to a pane.
func SendEnter(pane string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", pane, "Enter")
	if err := cmd.Run(); err != nil {
		return errors.Join(ErrCommandFailed, err)
	}

	return nil
}

package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds user preferences loaded from the config file.
type Config struct {
	// DisplayLocation is the timezone used to render rate-limit reset times
	// in the TUI. Defaults to time.Local when unset or invalid.
	DisplayLocation *time.Location
	// BellEnabled controls whether a terminal bell (\a) is emitted when
	// auto-continue sends the "continue" command. Defaults to false.
	BellEnabled bool
}

// Path returns the config file path, honoring $XDG_CONFIG_HOME.
func Path() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "tmux-claude-refresh", "config")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "tmux-claude-refresh", "config")
}

// Load reads the config file. Missing file is not an error: it returns a
// Config with DisplayLocation set to time.Local. An invalid timezone value
// is reported to stderr and falls back to time.Local.
func Load() Config {
	cfg := Config{DisplayLocation: time.Local}

	f, err := os.Open(Path())
	if err != nil {
		return cfg
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			continue
		}
		switch key {
		case "timezone":
			loc, err := time.LoadLocation(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "config: invalid timezone %q: %v (falling back to %s)\n",
					val, err, time.Local)
				continue
			}
			cfg.DisplayLocation = loc
		case "bell":
			cfg.BellEnabled = val == "on" || val == "true" || val == "yes" || val == "1"
		}
	}
	return cfg
}

// SaveTimezone writes `timezone = <name>` to the config file, creating the
// parent directory if needed. Existing keys are preserved; a `timezone` key,
// if present, is replaced in place.
func SaveTimezone(name string) error {
	path := Path()
	existing := readExisting(path)

	out := make([]string, 0, len(existing)+1)
	wrote := false
	for _, line := range existing {
		if key, _, ok := splitKV(line); ok && key == "timezone" {
			out = append(out, "timezone = "+name)
			wrote = true
			continue
		}
		out = append(out, line)
	}
	if !wrote {
		out = append(out, "timezone = "+name)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

// SaveBell writes `bell = on|off` to the config file, creating the
// parent directory if needed. Existing keys are preserved; a `bell` key,
// if present, is replaced in place.
func SaveBell(enabled bool) error {
	path := Path()
	existing := readExisting(path)

	val := "off"
	if enabled {
		val = "on"
	}

	out := make([]string, 0, len(existing)+1)
	wrote := false
	for _, line := range existing {
		if key, _, ok := splitKV(line); ok && key == "bell" {
			out = append(out, "bell = "+val)
			wrote = true
			continue
		}
		out = append(out, line)
	}
	if !wrote {
		out = append(out, "bell = "+val)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func splitKV(line string) (key, val string, ok bool) {
	if i := strings.IndexByte(line, '='); i >= 0 {
		key = strings.TrimSpace(line[:i])
		val = strings.TrimSpace(line[i+1:])
		// Strip optional surrounding quotes.
		val = strings.Trim(val, `"'`)
		return key, val, key != ""
	}
	return "", "", false
}

func readExisting(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	return lines
}

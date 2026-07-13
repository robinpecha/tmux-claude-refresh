package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg := Load()
	if cfg.DisplayLocation != time.Local {
		t.Errorf("expected time.Local when no config file, got %v", cfg.DisplayLocation)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := SaveTimezone("Europe/Prague"); err != nil {
		t.Fatalf("SaveTimezone: %v", err)
	}

	// File should exist at <dir>/tmux-claude-refresh/config.
	path := filepath.Join(dir, "tmux-claude-refresh", "config")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	cfg := Load()
	want, _ := time.LoadLocation("Europe/Prague")
	if cfg.DisplayLocation.String() != want.String() {
		t.Errorf("expected %s, got %s", want, cfg.DisplayLocation)
	}
}

func TestSaveTimezone_ReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := SaveTimezone("Europe/Prague"); err != nil {
		t.Fatal(err)
	}
	if err := SaveTimezone("Asia/Tokyo"); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.DisplayLocation.String() != "Asia/Tokyo" {
		t.Errorf("expected Asia/Tokyo, got %s", cfg.DisplayLocation)
	}
}

func TestLoad_InvalidTimezoneFallsBack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "tmux-claude-refresh", "config")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("timezone = Not/A/Real/Zone\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.DisplayLocation != time.Local {
		t.Errorf("expected fallback to time.Local, got %v", cfg.DisplayLocation)
	}
}

package tui

import (
	"strings"
	"testing"
)

func TestFilteredTimezones_Empty(t *testing.T) {
	got := filteredTimezones("")
	if len(got) != len(ianaTimezones) {
		t.Errorf("empty filter should return all %d zones, got %d", len(ianaTimezones), len(got))
	}
}

func TestFilteredTimezones_SubstringCaseInsensitive(t *testing.T) {
	got := filteredTimezones("prague")
	if len(got) == 0 {
		t.Fatal("expected at least one match for 'prague'")
	}
	for _, z := range got {
		if !strings.Contains(strings.ToLower(z), "prague") {
			t.Errorf("non-matching zone returned: %s", z)
		}
	}
	// Europe/Prague must be among the matches.
	found := false
	for _, z := range got {
		if z == "Europe/Prague" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Europe/Prague missing from 'prague' results")
	}
}

func TestFilteredTimezones_NoMatch(t *testing.T) {
	if got := filteredTimezones("zzz/notreal"); len(got) != 0 {
		t.Errorf("expected no matches, got %d", len(got))
	}
}

func TestRenderTZPicker_NoPanic(t *testing.T) {
	m := New("test", "", nil, false)
	m.width = 80
	m.height = 24
	m.showTZPicker = true
	m.tzFilter = "europe"
	m.tzIndex = 5

	out := m.renderTZPicker()
	if out == "" {
		t.Error("renderTZPicker returned empty string")
	}
	if !strings.Contains(out, "Europe/") {
		t.Error("expected picker to list a Europe/ zone for filter 'europe'")
	}
	if !strings.Contains(out, "❯") {
		t.Error("expected a selected row marker ❯")
	}
}

func TestRenderTZPicker_NoMatches(t *testing.T) {
	m := New("test", "", nil, false)
	m.width = 80
	m.height = 24
	m.showTZPicker = true
	m.tzFilter = "zzz/notreal"

	out := m.renderTZPicker()
	if !strings.Contains(out, "no timezones match") {
		t.Errorf("expected 'no timezones match' message, got: %s", out)
	}
}

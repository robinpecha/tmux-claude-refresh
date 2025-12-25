package detector

import (
	"testing"
	"time"
)

func TestDetectUsageLimit_OldFormat(t *testing.T) {
	// Use a timestamp in the future
	futureTime := time.Now().Add(1 * time.Hour).Unix()
	content := "Some output\nClaude AI usage limit reached|" + string(rune(futureTime)) + "\nMore output"

	// Actually, let's use a real string format
	content = "Some output\nClaude AI usage limit reached|1735200000\nMore output"

	info := DetectUsageLimit(content)

	if !info.Detected {
		t.Error("Expected limit to be detected")
	}
	if info.Format != "old" {
		t.Errorf("Expected format 'old', got %q", info.Format)
	}
	if info.ResetTime.Unix() != 1735200000 {
		t.Errorf("Expected reset time 1735200000, got %d", info.ResetTime.Unix())
	}
}

func TestDetectUsageLimit_NewFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantHour int
	}{
		{
			name:     "morning reset",
			content:  "Usage limit reached ∙ resets 9am",
			wantHour: 9,
		},
		{
			name:     "afternoon reset",
			content:  "limit reached ∙ resets 3pm",
			wantHour: 15,
		},
		{
			name:     "noon reset",
			content:  "limit reached ∙ resets 12pm",
			wantHour: 12,
		},
		{
			name:     "midnight reset",
			content:  "limit reached ∙ resets 12am",
			wantHour: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectUsageLimit(tt.content)

			if !info.Detected {
				t.Error("Expected limit to be detected")
			}
			if info.Format != "new" {
				t.Errorf("Expected format 'new', got %q", info.Format)
			}
			if info.ResetTime.Hour() != tt.wantHour {
				t.Errorf("Expected hour %d, got %d", tt.wantHour, info.ResetTime.Hour())
			}
		})
	}
}

func TestDetectUsageLimit_NoMatch(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "normal output",
			content: "Hello, I'm Claude. How can I help you today?",
		},
		{
			name:    "partial match",
			content: "limit reached but no time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectUsageLimit(tt.content)

			if info.Detected {
				t.Error("Expected no limit to be detected")
			}
		})
	}
}

func TestParseOldFormat(t *testing.T) {
	timestamp := int64(1735200000)
	result, err := parseOldFormat("1735200000")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Unix() != timestamp {
		t.Errorf("Expected %d, got %d", timestamp, result.Unix())
	}
}

func TestParseOldFormat_Invalid(t *testing.T) {
	_, err := parseOldFormat("notanumber")

	if err == nil {
		t.Error("Expected error for invalid input")
	}
}

func TestParseNewFormat(t *testing.T) {
	tests := []struct {
		hour     string
		period   string
		wantHour int
	}{
		{"9", "am", 9},
		{"12", "am", 0},
		{"12", "pm", 12},
		{"3", "pm", 15},
		{"11", "pm", 23},
	}

	for _, tt := range tests {
		t.Run(tt.hour+tt.period, func(t *testing.T) {
			result, err := parseNewFormat(tt.hour, tt.period)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Hour() != tt.wantHour {
				t.Errorf("Expected hour %d, got %d", tt.wantHour, result.Hour())
			}
		})
	}
}

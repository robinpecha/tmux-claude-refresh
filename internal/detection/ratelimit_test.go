package detection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return string(data)
}

func TestCheckRateLimit_NewFormat(t *testing.T) {
	content := loadFixture(t, "rate_limit_new_format.txt")
	// Fixture contains "(Europe/London)"; display in that same zone so the
	// rendered time is a deterministic no-op regardless of DST.
	london, _ := time.LoadLocation("Europe/London")
	status := CheckRateLimit(content, london)

	if !status.IsLimited {
		t.Error("expected IsLimited to be true")
	}
	if status.ResetsAt != "10pm (Europe/London)" {
		t.Errorf("expected ResetsAt to be '10pm (Europe/London)', got '%s'", status.ResetsAt)
	}
	if status.ResetTime.IsZero() {
		t.Error("expected ResetTime to be set")
	}
}

func TestCheckRateLimit_OldFormat(t *testing.T) {
	content := loadFixture(t, "rate_limit_old_format.txt")
	status := CheckRateLimit(content, time.Local)

	if !status.IsLimited {
		t.Error("expected IsLimited to be true")
	}
	if status.ResetsAt != "2pm (Local)" {
		t.Errorf("expected ResetsAt to be '2pm (Local)', got '%s'", status.ResetsAt)
	}
	if status.ResetTime.IsZero() {
		t.Error("expected ResetTime to be set")
	}
}

func TestCheckRateLimit_NoMatch(t *testing.T) {
	content := loadFixture(t, "not_claude_code.txt")
	status := CheckRateLimit(content, time.Local)

	if status.IsLimited {
		t.Error("expected IsLimited to be false")
	}
}

func TestCheckRateLimit_TimeFormats(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	dublin, _ := time.LoadLocation("Europe/Dublin")

	cases := []struct {
		name       string
		content    string
		wantTime   string
		displayLoc *time.Location // nil -> time.Local
	}{
		{
			name:     "simple pm",
			content:  "You've hit your limit · resets 2pm",
			wantTime: "2pm (Local)",
		},
		{
			name:     "simple am",
			content:  "You've hit your limit · resets 9am",
			wantTime: "9am (Local)",
		},
		{
			name:     "with minutes",
			content:  "limit reached ∙ resets 10:30am",
			wantTime: "10:30am (Local)",
		},
		{
			name:     "with space before am/pm",
			content:  "limit reached ∙ resets 3 pm",
			wantTime: "3pm (Local)",
		},
		{
			name:       "double digit hour",
			content:    "You've hit your limit · resets 11pm (Europe/London)",
			wantTime:   "11pm (Europe/London)",
			displayLoc: london,
		},
		{
			name:       "session limit with clock time",
			content:    "You've hit your session limit · resets 2:20pm (Europe/Dublin)",
			wantTime:   "2:20pm (Europe/Dublin)",
			displayLoc: dublin,
		},
		{
			name:     "weekly limit",
			content:  "You've hit your weekly limit · resets 9am",
			wantTime: "9am (Local)",
		},
		{
			name:       "extra usage",
			content:    "You're out of extra usage · resets 8pm (Europe/London)",
			wantTime:   "8pm (Europe/London)",
			displayLoc: london,
		},
		{
			name:     "session limit minutes format",
			content:  "You've hit your session limit · resets 45m",
			wantTime: "45m",
		},
		{
			name:     "extra usage minutes format",
			content:  "You're out of extra usage · resets 8m",
			wantTime: "8m",
		},
		{
			name:     "minutes remaining format",
			content:  "⚠ Limit reached (resets 8m)",
			wantTime: "8m",
		},
		{
			name:     "minutes remaining double digit",
			content:  "Limit reached (resets 45m)",
			wantTime: "45m",
		},
		{
			name:     "minutes remaining triple digit",
			content:  "⚠ Limit reached (resets 120m)",
			wantTime: "120m",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc := tc.displayLoc
			if loc == nil {
				loc = time.Local
			}
			status := CheckRateLimit(tc.content, loc)
			if !status.IsLimited {
				t.Error("expected IsLimited to be true")
			}
			if status.ResetsAt != tc.wantTime {
				t.Errorf("expected ResetsAt to be '%s', got '%s'", tc.wantTime, status.ResetsAt)
			}
		})
	}
}

func TestCheckRateLimit_MinutesFormat(t *testing.T) {
	status := CheckRateLimit("⚠ Limit reached (resets 30m)", time.Local)

	if !status.IsLimited {
		t.Error("expected IsLimited to be true")
	}
	if status.ResetsAt != "30m" {
		t.Errorf("expected ResetsAt to be '30m', got '%s'", status.ResetsAt)
	}
	if status.ResetTime.IsZero() {
		t.Error("expected ResetTime to be set")
	}
	// TimeUntil should be approximately 30 minutes (within 1 second tolerance)
	expectedDuration := 30 * time.Minute
	if status.TimeUntil < expectedDuration-time.Second || status.TimeUntil > expectedDuration+time.Second {
		t.Errorf("expected TimeUntil to be ~30m, got %v", status.TimeUntil)
	}
}

func TestCheckRateLimit_FallbackNoTime(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{
			name:    "hit your limit without time",
			content: "You've hit your limit",
		},
		{
			name:    "hit your limit with curly apostrophe",
			content: "You've hit your limit",
		},
		{
			name:    "hit your session limit",
			content: "You've hit your session limit",
		},
		{
			name:    "hit your weekly limit",
			content: "You've hit your weekly limit",
		},
		{
			name:    "out of extra usage",
			content: "You're out of extra usage",
		},
		{
			name:    "limit reached without time",
			content: "Limit reached - please wait",
		},
		{
			name:    "rate limited status",
			content: "⚠ Rate limited",
		},
		{
			name:    "limit reached with unparseable time format",
			content: "Limit reached (resets in 2 hours)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := CheckRateLimit(tc.content, time.Local)
			if !status.IsLimited {
				t.Error("expected IsLimited to be true")
			}
			if status.ResetsAt != "" {
				t.Errorf("expected ResetsAt to be empty for fallback, got '%s'", status.ResetsAt)
			}
			if !status.ResetTime.IsZero() {
				t.Error("expected ResetTime to be zero for fallback")
			}
		})
	}
}

func TestCheckRateLimit_NoMatchCases(t *testing.T) {
	cases := []string{
		"Normal output without rate limit",
		"The limit of my patience",
		"Rate your experience",
	}

	for _, content := range cases {
		t.Run(content, func(t *testing.T) {
			status := CheckRateLimit(content, time.Local)
			if status.IsLimited {
				t.Errorf("expected IsLimited to be false for: %q", content)
			}
		})
	}
}

func TestHasReset(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name   string
		status RateLimitStatus
		want   bool
	}{
		{
			name:   "not limited",
			status: RateLimitStatus{IsLimited: false},
			want:   false,
		},
		{
			name:   "limited but no reset time",
			status: RateLimitStatus{IsLimited: true},
			want:   false,
		},
		{
			name: "limited, reset time in future",
			status: RateLimitStatus{
				IsLimited: true,
				ResetTime: now.Add(1 * time.Hour),
			},
			want: false,
		},
		{
			name: "limited, reset time in past",
			status: RateLimitStatus{
				IsLimited: true,
				ResetTime: now.Add(-1 * time.Hour),
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.status.HasReset()
			if got != tc.want {
				t.Errorf("HasReset() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCheckRateLimit_TimezoneConversion exercises parsing the source timezone
// Claude embeds in "(...)" and converting the reset instant into a different
// display timezone. Uses time.FixedZone for fully deterministic offsets (no
// reliance on the host's zone or DST).
func TestCheckRateLimit_TimezoneConversion(t *testing.T) {
	// Source is UTC (10pm UTC). Display is a fixed UTC+3 zone.
	// 10pm UTC -> 1am the following day in UTC+3.
	utcPlus3 := time.FixedZone("Display+03", 3*60*60)
	status := CheckRateLimit("You've hit your limit · resets 10pm (UTC)", utcPlus3)

	if !status.IsLimited {
		t.Fatal("expected IsLimited to be true")
	}
	if status.ResetsAt != "1am (Display+03)" {
		t.Errorf("expected ResetsAt '1am (Display+03)', got '%s'", status.ResetsAt)
	}
	// ResetTime is an absolute instant; in UTC it should be 22:00.
	if rt := status.ResetTime.In(time.UTC); rt.Hour() != 22 || rt.Minute() != 0 {
		t.Errorf("expected ResetTime 22:00 UTC, got %v", rt)
	}
}

// TestCheckRateLimit_NoSourceTimezone confirms that when Claude omits the
// "(...)" suffix, the source defaults to the host local time (so the reset
// instant is anchored in Local) and the display reflects the passed location.
func TestCheckRateLimit_NoSourceTimezone(t *testing.T) {
	prague, _ := time.LoadLocation("Europe/Prague")
	status := CheckRateLimit("You've hit your session limit · resets 9am", prague)

	if !status.IsLimited {
		t.Fatal("expected IsLimited to be true")
	}
	// 9am Local displayed in Prague. Since the source defaults to Local, the
	// rendered time is 9am shifted by (Prague offset - Local offset); the exact
	// hour depends on the host, but the zone label must be Prague.
	wantSuffix := "(Europe/Prague)"
	if !strings.HasSuffix(status.ResetsAt, wantSuffix) {
		t.Errorf("expected ResetsAt to end with %q, got '%s'", wantSuffix, status.ResetsAt)
	}
	if status.ResetTime.IsZero() {
		t.Error("expected ResetTime to be set")
	}
}

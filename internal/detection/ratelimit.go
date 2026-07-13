package detection

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RateLimitStatus represents the rate limit state of a pane
type RateLimitStatus struct {
	IsLimited  bool
	ResetsAt   string    // Original string like "2pm" or "10:30am"
	ResetTime  time.Time // Parsed reset time
	TimeUntil  time.Duration
}

// Rate limit patterns - multiple formats Claude Code uses
// Examples: "limit reached ∙ resets 2pm", "limit reached ∙ resets 10:30am"
//           "You've hit your limit · resets 10pm (Europe/London)"
//           "You've hit your session limit · resets 2:20pm (Europe/Dublin)"
//           "You've hit your weekly limit · resets 9am"
//           "You're out of extra usage · resets 8pm (Europe/London)"
//           "Limit reached (resets 8m)" - minutes remaining format
//
// The `(?:\w+\s+)?` allows an optional qualifier word that Claude Code now
// inserts between "your" and "limit" (e.g. "session", "weekly").
// The trailing `(?:\s*\(([^)]+)\))?` captures the optional source timezone
// Claude embeds in parentheses, e.g. "(Europe/London)".
var rateLimitPatterns = []*regexp.Regexp{
	// New format: "You've hit your [session|weekly] limit · resets 10pm (Europe/London)"
	regexp.MustCompile(`(?i)hit\s+your\s+(?:\w+\s+)?limit.*resets?\s+(\d{1,2}(?::\d{2})?\s*[ap]m)(?:\s*\(([^)]+)\))?`),
	// New format (extra usage): "You're out of extra usage · resets 8pm (Europe/London)"
	regexp.MustCompile(`(?i)you're\s+out\s+of\s+extra\s+usage.*resets?\s+(\d{1,2}(?::\d{2})?\s*[ap]m)(?:\s*\(([^)]+)\))?`),
	// Original format: "limit reached ∙ resets 2pm"
	regexp.MustCompile(`(?i)limit\s+reached.*resets?\s+(\d{1,2}(?::\d{2})?\s*[ap]m)(?:\s*\(([^)]+)\))?`),
	// Minutes remaining format: "Limit reached (resets 8m)" or "resets 45m"
	regexp.MustCompile(`(?i)(?:hit\s+your\s+(?:\w+\s+)?limit|you're\s+out\s+of\s+extra\s+usage|limit\s+reached).*resets?\s+(\d{1,3})m\b`),
}

// Fallback patterns - detect rate limit without capturing time
// Used when we can't parse a specific reset time
// These patterns are more specific to avoid false positives
var rateLimitFallbackPatterns = []*regexp.Regexp{
	// "You've hit your [session] limit" - Claude Code's primary message
	// `(?:\w+\s+)?` also matches "your session limit" / "your weekly limit"
	regexp.MustCompile(`(?i)you['']ve\s+hit\s+your\s+(?:\w+\s+)?limit`),
	// "You're out of extra usage"
	regexp.MustCompile(`(?i)you're\s+out\s+of\s+extra\s+usage`),
	// "Limit reached" at word boundary (not "rate limit exceeded" or similar)
	regexp.MustCompile(`(?i)\blimit\s+reached\b`),
	// "rate limited" as a status indicator
	regexp.MustCompile(`(?i)\brate\s+limited\b`),
}

// CheckRateLimit checks pane content for rate limit messages. displayLoc is
// the timezone used to render the reset time for the user; the underlying
// ResetTime instant is computed in the timezone Claude embedded in the
// message (falling back to the host's local time when Claude omits one).
func CheckRateLimit(content string, displayLoc *time.Location) RateLimitStatus {
	if displayLoc == nil {
		displayLoc = time.Local
	}
	// Try patterns that capture reset time first
	var match []string
	var patternIdx int
	for i, pattern := range rateLimitPatterns {
		match = pattern.FindStringSubmatch(content)
		if match != nil {
			patternIdx = i
			break
		}
	}

	// If no time-capturing pattern matched, try fallback patterns
	if match == nil {
		for _, pattern := range rateLimitFallbackPatterns {
			if pattern.MatchString(content) {
				// Rate limited but couldn't parse time - return with empty ResetsAt
				return RateLimitStatus{
					IsLimited: true,
					ResetsAt:  "", // Unknown reset time
				}
			}
		}
		return RateLimitStatus{IsLimited: false}
	}

	now := time.Now()

	// The last pattern is the minutes-remaining format (e.g., "8m" -> "8")
	if patternIdx == len(rateLimitPatterns)-1 {
		resetStr := match[1]
		minutes, err := strconv.Atoi(resetStr)
		if err != nil {
			return RateLimitStatus{
				IsLimited: true,
				ResetsAt:  resetStr + "m",
			}
		}
		resetTime := now.Add(time.Duration(minutes) * time.Minute)
		return RateLimitStatus{
			IsLimited: true,
			ResetsAt:  resetStr + "m",
			ResetTime: resetTime,
			TimeUntil: time.Duration(minutes) * time.Minute,
		}
	}

	// Clock time format (e.g., "8pm", "10:30am")
	resetStr := match[1]
	// Optional source timezone (group 2); empty when Claude omits "(...)".
	sourceLoc := time.Local
	if len(match) >= 3 && match[2] != "" {
		if loc, err := time.LoadLocation(match[2]); err == nil {
			sourceLoc = loc
		}
	}

	resetTime, err := parseResetTime(resetStr, sourceLoc, now)
	if err != nil {
		// Pattern matched but couldn't parse time - still rate limited
		return RateLimitStatus{
			IsLimited: true,
			ResetsAt:  resetStr,
		}
	}

	timeUntil := resetTime.Sub(now)

	// If the time is more than 1 hour in the past, it's likely for tomorrow.
	// But if it's within the last hour, keep it as-is so we can detect
	// that the reset time has passed and trigger the continue action.
	if timeUntil < -1*time.Hour {
		resetTime = resetTime.Add(24 * time.Hour)
		timeUntil = resetTime.Sub(now)
	}

	return RateLimitStatus{
		IsLimited: true,
		ResetsAt:  formatResetDisplay(resetTime, displayLoc),
		ResetTime: resetTime,
		TimeUntil: timeUntil,
	}
}

// formatResetDisplay renders the reset instant in the user's display timezone
// as e.g. "11pm (Europe/Prague)", omitting minutes when zero to match Claude's
// natural style.
func formatResetDisplay(t time.Time, displayLoc *time.Location) string {
	tt := t.In(displayLoc)
	var s string
	if tt.Minute() == 0 {
		s = tt.Format("3pm")
	} else {
		s = tt.Format("3:04pm")
	}
	return s + " (" + displayLoc.String() + ")"
}

// parseResetTime parses a clock time string like "2pm" or "10:30am" into a
// time.Time for the current date in loc, anchored against now (the date
// components are taken from now in loc so midnight-boundary cases are handled).
func parseResetTime(s string, loc *time.Location, now time.Time) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	s = strings.ToLower(strings.TrimSpace(s))
	nowIn := now.In(loc)

	// Try parsing with minutes first: "10:30am"
	formats := []string{
		"3:04pm",
		"3:04 pm",
		"3pm",
		"3 pm",
	}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, s, loc)
		if err == nil {
			// Combine parsed time with today's date (in loc)
			return time.Date(nowIn.Year(), nowIn.Month(), nowIn.Day(),
				t.Hour(), t.Minute(), 0, 0, loc), nil
		}
	}

	// Manual parsing as fallback
	isPM := strings.Contains(s, "pm")
	s = strings.ReplaceAll(s, "am", "")
	s = strings.ReplaceAll(s, "pm", "")
	s = strings.TrimSpace(s)

	var hour, minute int
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		hour, _ = strconv.Atoi(parts[0])
		minute, _ = strconv.Atoi(parts[1])
	} else {
		hour, _ = strconv.Atoi(s)
		minute = 0
	}

	// Convert to 24-hour format
	if isPM && hour != 12 {
		hour += 12
	} else if !isPM && hour == 12 {
		hour = 0
	}

	return time.Date(nowIn.Year(), nowIn.Month(), nowIn.Day(),
		hour, minute, 0, 0, loc), nil
}

// HasReset checks if the rate limit has reset (time has passed)
func (r RateLimitStatus) HasReset() bool {
	if !r.IsLimited {
		return false
	}
	if r.ResetTime.IsZero() {
		return false
	}
	return time.Now().After(r.ResetTime)
}

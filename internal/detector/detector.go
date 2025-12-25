package detector

import (
	"regexp"
	"strconv"
	"time"
)

// LimitInfo contains information about a detected usage limit.
type LimitInfo struct {
	Detected   bool
	ResetTime  time.Time
	RawMessage string
	Format     string // "old" or "new"
}

var (
	// Old format: Claude AI usage limit reached|<timestamp>
	oldFormatPattern = regexp.MustCompile(`Claude AI usage limit reached\|(\d+)`)

	// New format: limit reached ∙ resets Xam/pm
	newFormatPattern = regexp.MustCompile(`limit reached.*resets (\d{1,2})(am|pm)`)
)

// DetectUsageLimit checks content for usage limit messages and parses the reset time.
func DetectUsageLimit(content string) *LimitInfo {
	// Try old format first
	if matches := oldFormatPattern.FindStringSubmatch(content); len(matches) >= 2 {
		resetTime, err := parseOldFormat(matches[1])
		if err == nil {
			return &LimitInfo{
				Detected:   true,
				ResetTime:  resetTime,
				RawMessage: matches[0],
				Format:     "old",
			}
		}
	}

	// Try new format
	if matches := newFormatPattern.FindStringSubmatch(content); len(matches) >= 3 {
		resetTime, err := parseNewFormat(matches[1], matches[2])
		if err == nil {
			return &LimitInfo{
				Detected:   true,
				ResetTime:  resetTime,
				RawMessage: matches[0],
				Format:     "new",
			}
		}
	}

	return &LimitInfo{Detected: false}
}

// parseOldFormat parses a Unix timestamp string.
func parseOldFormat(timestampStr string) (time.Time, error) {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

// parseNewFormat parses hour and am/pm to determine the next reset time.
func parseNewFormat(hourStr, period string) (time.Time, error) {
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return time.Time{}, err
	}

	// Convert to 24-hour format
	if period == "pm" && hour != 12 {
		hour += 12
	} else if period == "am" && hour == 12 {
		hour = 0
	}

	now := time.Now()
	resetTime := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())

	// If reset time has passed today, it's tomorrow
	if resetTime.Before(now) {
		resetTime = resetTime.Add(24 * time.Hour)
	}

	return resetTime, nil
}

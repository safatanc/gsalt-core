package pkg

import "time"

// ParseFlipExpiryTime parses the date string from Flip's API into a *time.Time object.
// Flip's V2 API might return dates in "2006-01-02 15:04:05" format.
func ParseFlipExpiryTime(dateStr *string) *time.Time {
	if dateStr == nil || *dateStr == "" {
		return nil
	}

	// Flip's documentation for v2 mentions "YYYY-MM-DD HH:mm:ss"
	const layout = "2006-01-02 15:04:05"
	t, err := time.Parse(layout, *dateStr)
	if err != nil {
		// You might want to log this error, but for now, we return nil
		return nil
	}
	return &t
}

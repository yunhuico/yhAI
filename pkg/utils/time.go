package utils

import "time"

// NowHumanDurationFrom calc duration from given time for human, truncate 10*time.Millisecond
func NowHumanDurationFrom(t time.Time) time.Duration {
	return time.Since(t).Truncate(1 * time.Millisecond)
}

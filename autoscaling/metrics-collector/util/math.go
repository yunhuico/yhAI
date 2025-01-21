package util

// Round returns nearest interger to a float value.
// It works like Math.round() function in JavaScript
func Round(val float64) int {
	if val < 0 {
		return int(val - 0.5)
	}
	return int(val + 0.5)
}

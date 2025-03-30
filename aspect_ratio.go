package main

import "math"

func GetVideoAspectRatio(width, height int) string {
	if height < width {
		ratio := math.Round(float64(height) / float64(width) * 100.0)

		if ratio - 178 <= 0 {
			return "16:9"
		} else {
			return "other"
		}
	} else if height > width {
		ratio := math.Round(float64(width) / float64(height) * 100.0)
		if ratio - 563 <= 0 {
			return "9:16"
		} else {
			return "other"
		}
	}

	return "other"
}
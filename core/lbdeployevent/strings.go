package lbdeployevent

import (
	"fmt"
	"time"
)

func plural[T ~int | ~int64](value T, singular, plural string) string {
	if value == 1 {
		return singular
	}
	return plural
}

func bitrate(transferred int64, duration time.Duration) string {
	if transferred == 0 || duration == 0 {
		return "0"
	}
	const mebibit = float64(1048576)
	const conversion = (8 / mebibit)
	bytesPerSecond := float64(transferred) / duration.Seconds()
	return fmt.Sprintf("%.02f", bytesPerSecond*conversion)
}

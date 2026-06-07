package internal

import (
	"math"
	"time"
)

func expLerp(current, target, rate float32, dt time.Duration, enabled bool) float32 {
	if !enabled {
		return target
	}
	seconds := float32(dt.Seconds())
	if seconds <= 0 {
		return current
	}
	alpha := 1 - float32(math.Exp(float64(-rate*seconds)))
	return current + (target-current)*alpha
}

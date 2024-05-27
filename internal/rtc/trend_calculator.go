package rtc

import (
	"math"
	"time"
)

type TrendCalculator struct {
	value                 uint32
	highestValue          uint32
	highestValueUpdatedAt time.Time
	decreaseFactor        float64
}

func NewTrendCalculator(decreaseFactor float64) *TrendCalculator {
	return &TrendCalculator{
		decreaseFactor: decreaseFactor,
	}
}

func (tc *TrendCalculator) Update(value uint32, now time.Time) {
	if tc.value == 0 {
		tc.value = value
		tc.highestValue = value
		tc.highestValueUpdatedAt = now
		return
	}

	// If new value is bigger or equal than current one, use it.
	if value >= tc.value {
		tc.value = value
		tc.highestValue = value
		tc.highestValueUpdatedAt = now
	} else {
		// Otherwise decrease current value.
		elapsed := now.Sub(tc.highestValueUpdatedAt).Seconds()
		subtraction := uint32(float64(tc.highestValue) * tc.decreaseFactor * elapsed)
		if tc.highestValue > subtraction {
			tc.value = uint32(math.Max(float64(value), float64(tc.highestValue-subtraction)))
		} else {
			tc.value = value
		}
	}
}

func (tc *TrendCalculator) ForceUpdate(value uint32, now time.Time) {
	tc.value = value
	tc.highestValue = value
	tc.highestValueUpdatedAt = now
}

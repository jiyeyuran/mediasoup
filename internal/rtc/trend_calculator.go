package rtc

import (
	"math"
)

const DefaultDecreaseFactor float64 = 0.05

func WithDecreaseFactor(decreaseFactor float64) func(*TrendCalculator) {
	return func(tc *TrendCalculator) {
		tc.decreaseFactor = decreaseFactor
	}
}

type TrendCalculator struct {
	value                   uint32
	highestValue            uint32
	highestValueUpdatedAtMs uint64
	decreaseFactor          float64
}

func NewTrendCalculator(options ...func(*TrendCalculator)) *TrendCalculator {
	tc := &TrendCalculator{
		decreaseFactor: DefaultDecreaseFactor,
	}
	for _, option := range options {
		option(tc)
	}
	return tc
}

func (tc *TrendCalculator) Update(value uint32, nowMs uint64) {
	if tc.value == 0 {
		tc.value = value
		tc.highestValue = value
		tc.highestValueUpdatedAtMs = nowMs
		return
	}

	// If new value is bigger or equal than current one, use it.
	if value >= tc.value {
		tc.value = value
		tc.highestValue = value
		tc.highestValueUpdatedAtMs = nowMs
	} else {
		// Otherwise decrease current value.
		elapsedMs := nowMs - tc.highestValueUpdatedAtMs
		subtraction := uint32(float64(tc.highestValue) * tc.decreaseFactor * float64(elapsedMs) / 1000)
		if tc.highestValue > subtraction {
			tc.value = uint32(math.Max(float64(value), float64(tc.highestValue-subtraction)))
		} else {
			tc.value = value
		}
	}
}

func (tc *TrendCalculator) ForceUpdate(value uint32, nowMs uint64) {
	tc.value = value
	tc.highestValue = value
	tc.highestValueUpdatedAtMs = nowMs
}

func (tc TrendCalculator) GetValue() uint32 {
	return tc.value
}

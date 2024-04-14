package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type data struct {
	offset uint32
	size   uint64
	rate   uint32
}

func validate(t *testing.T, rate *rateCalculator, timeBase uint64, input []data) {
	for _, item := range input {
		rate.Update(item.size, timeBase+uint64(item.offset))
		rateValue := rate.GetRate(timeBase + uint64(item.offset))

		if rateValue != item.rate {
			t.Errorf("Rate does not match: got %v, want %v", rateValue, item.rate)
		}
	}
}

func TestRateCalculator(t *testing.T) {
	nowMs := getTimeMs()

	testCases := []struct {
		name  string
		rate  *rateCalculator
		input []data
		after func(t *testing.T, r *rateCalculator)
	}{
		{
			"receive single item per 1000 ms",
			newRateCalculator(1000, 8000, 100),
			[]data{
				{0, 5, 40},
			},
			nil,
		},
		{
			"receive multiple items per 1000 ms",
			newRateCalculator(1000, 8000, 100),
			[]data{
				{0, 5, 40},
				{100, 2, 56},
				{300, 2, 72},
				{999, 4, 104},
			},
			nil,
		},
		{
			"receive item every 1000 ms",
			newRateCalculator(1000, 8000, 100),
			[]data{
				{0, 5, 40},
				{1000, 5, 40},
				{2000, 5, 40},
			},
			nil,
		},
		{
			"slide",
			newRateCalculator(1000, 8000, 1000),
			[]data{
				{0, 5, 40},
				{999, 2, 56},
				{1001, 1, 24},
				{1001, 1, 32},
				{2000, 1, 24},
			},
			func(t *testing.T, r *rateCalculator) {
				assert.Zero(t, r.GetRate(nowMs+3001))
			},
		},
		{
			"slide with 100 items",
			newRateCalculator(1000, 8000, 100),
			[]data{
				{0, 5, 40},
				{999, 2, 56},
				{1001, 1, 24}, // merged inside 999
				{1001, 1, 32}, // merged inside 999
				{2000, 1, 8},  // it will erase the item with timestamp=999,
				// removing also the next two samples.
				// The end estimation will include only the last sample.
			},
			func(t *testing.T, r *rateCalculator) {
				assert.Zero(t, r.GetRate(nowMs+3001))
			},
		},
		{
			"wrap",
			newRateCalculator(1000, 8000, 5),
			[]data{
				{1000, 1, 1 * 8},
				{1200, 1, 1*8 + 1*8},
				{1400, 1, 1*8 + 2*8},
				{1600, 1, 1*8 + 3*8},
				{1800, 1, 1*8 + 4*8},
				{2000, 1, 1*8 + (5-1)*8}, // starts wrap here
				{2200, 1, 1*8 + (6-2)*8},
				{2400, 1, 1*8 + (7-3)*8},
				{2600, 1, 1*8 + (8-4)*8},
				{2800, 1, 1*8 + (9-5)*8},
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validate(t, tc.rate, nowMs, tc.input)
			if tc.after != nil {
				tc.after(t, tc.rate)
			}
		})
	}
}

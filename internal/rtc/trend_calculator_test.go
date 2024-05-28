package rtc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrendCalculator(t *testing.T) {
	t.Run("trend values after same elapsed time match", func(t *testing.T) {
		trendA := NewTrendCalculator()
		trendB := NewTrendCalculator()

		trendA.Update(1000, 0)
		trendB.Update(1000, 0)

		require.Equal(t, uint32(1000), trendA.GetValue())
		require.Equal(t, uint32(1000), trendB.GetValue())

		trendA.Update(200, 500)
		trendA.Update(200, 1000)
		trendB.Update(200, 1000)

		require.Equal(t, trendA.GetValue(), trendB.GetValue())

		trendA.Update(200, 2000)
		trendA.Update(200, 4000)
		trendB.Update(200, 4000)

		require.Equal(t, trendA.GetValue(), trendB.GetValue())

		trendA.Update(2000, 5000)
		trendB.Update(2000, 5000)

		require.Equal(t, uint32(2000), trendA.GetValue())
		require.Equal(t, uint32(2000), trendB.GetValue())

		trendA.ForceUpdate(0, 5500)
		trendB.ForceUpdate(100, 5000)

		require.Equal(t, uint32(0), trendA.GetValue())
		require.Equal(t, uint32(100), trendB.GetValue())
	})
}

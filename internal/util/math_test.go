package util

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxOf(t *testing.T) {
	assert.EqualValues(t, math.MaxInt8, MaxOf[int8]())
	assert.EqualValues(t, math.MaxUint8, MaxOf[uint8]())
	assert.EqualValues(t, math.MaxInt16, MaxOf[int16]())
	assert.EqualValues(t, math.MaxUint16, MaxOf[uint16]())
	assert.EqualValues(t, math.MaxInt32, MaxOf[int32]())
	assert.EqualValues(t, math.MaxUint32, MaxOf[uint32]())
	assert.EqualValues(t, math.MaxInt64, MaxOf[int64]())
	assert.EqualValues(t, uint64(math.MaxUint64), MaxOf[uint64]())
}

func TestMinOf(t *testing.T) {
	assert.EqualValues(t, math.MinInt8, MinOf[int8]())
	assert.EqualValues(t, 0, MinOf[uint8]())
	assert.EqualValues(t, math.MinInt16, MinOf[int16]())
	assert.EqualValues(t, 0, MinOf[uint16]())
	assert.EqualValues(t, math.MinInt32, MinOf[int32]())
	assert.EqualValues(t, 0, MinOf[uint32]())
	assert.EqualValues(t, math.MinInt64, MinOf[int64]())
	assert.EqualValues(t, 0, MinOf[uint64]())
}

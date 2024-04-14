package rtc

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/constraints"
)

const MaxNumberFor15Bits = (1 << 15) - 1

type TestSeqManagerInput[T constraints.Unsigned] struct {
	input    T
	output   T
	sync     bool
	drop     bool
	maxInput int64
}

func validate[T constraints.Unsigned](t *testing.T, seqManager *SeqManager[T], inputs []*TestSeqManagerInput[T]) {
	for _, element := range inputs {
		if element.sync {
			seqManager.Sync(element.input - 1)
		}

		if element.drop {
			seqManager.Drop(element.input)
		} else {
			output, _ := seqManager.Input(element.input)

			assert.Equal(t, int(element.output), int(output), "input: %d", element.input)

			if element.maxInput != -1 {
				assert.Equal(t, int(element.maxInput), int(seqManager.GetMaxInput()), "input: %d", element.input)
			}
		}
	}
}

func TestIsSeqHigherThan(t *testing.T) {
	t.Run("0 is greater than 65000", func(t *testing.T) {
		assert.True(t, IsSeqHigherThan[uint16](0, 65000))
	})

	t.Run("0 is greater than 32500 in range 15", func(t *testing.T) {
		assert.True(t, IsSeqHigherThan[uint16](0, 32500, 1<<15-1))
	})
}

func TestSeqManager(t *testing.T) {
	t.Run("receive ordered numbers, no sync, no drop", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{2, 2, false, false, -1},
			{3, 3, false, false, -1},
			{4, 4, false, false, -1},
			{5, 5, false, false, -1},
			{6, 6, false, false, -1},
			{7, 7, false, false, -1},
			{8, 8, false, false, -1},
			{9, 9, false, false, -1},
			{10, 10, false, false, -1},
			{11, 11, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("receive ordered numbers, sync, no drop", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{2, 2, false, false, -1},
			{80, 3, true, false, -1},
			{81, 4, false, false, -1},
			{82, 5, false, false, -1},
			{83, 6, false, false, -1},
			{84, 7, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("receive ordered numbers, sync, drop", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{2, 2, false, false, -1},
			{3, 3, false, false, -1},
			{4, 4, true, false, -1}, // sync.
			{5, 5, false, false, -1},
			{6, 6, false, false, -1},
			{7, 7, true, false, -1}, // sync.
			{8, 0, false, true, -1}, // drop.
			{9, 8, false, false, -1},
			{11, 0, false, true, -1}, // drop.
			{10, 9, false, false, -1},
			{12, 10, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("receive ordered wrapped numbers", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{65533, 65533, false, false, -1},
			{65534, 65534, false, false, -1},
			{65535, 65535, false, false, -1},
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		validate(t, seqManager, inputs)
	})

	t.Run("receive sequence numbers with a big jump", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{1000, 1000, false, false, -1},
			{1001, 1001, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("receive mixed numbers with a big jump, drop before jump", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 0, false, true, -1}, // drop.
			{100, 99, false, false, -1},
			{100, 99, false, false, -1},
			{103, 0, false, true, -1}, // drop.
			{101, 100, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("receive mixed numbers with a big jump, drop after jump", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{100, 0, false, true, -1}, // drop.
			{103, 0, false, true, -1}, // drop.
			{101, 100, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("drop, receive numbers newer and older than the one dropped", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{2, 0, false, true, -1}, // drop.
			{3, 2, false, false, -1},
			{4, 3, false, false, -1},
			{1, 1, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("receive mixed numbers, sync, drop", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{2, 2, false, false, -1},
			{3, 3, false, false, -1},
			{7, 7, false, false, -1},
			{6, 0, false, true, -1}, // drop.
			{8, 8, false, false, -1},
			{10, 10, false, false, -1},
			{9, 9, false, false, -1},
			{11, 11, false, false, -1},
			{0, 12, true, false, -1}, // sync.
			{2, 14, false, false, -1},
			{3, 15, false, false, -1},
			{4, 16, false, false, -1},
			{5, 17, false, false, -1},
			{6, 18, false, false, -1},
			{7, 19, false, false, -1},
			{8, 20, false, false, -1},
			{9, 21, false, false, -1},
			{10, 22, false, false, -1},
			{9, 0, false, true, -1},   // drop.
			{61, 23, true, false, -1}, // sync.
			{62, 24, false, false, -1},
			{63, 25, false, false, -1},
			{64, 26, false, false, -1},
			{65, 27, false, false, -1},
			{11, 28, true, false, -1}, // sync.
			{12, 29, false, false, -1},
			{13, 30, false, false, -1},
			{14, 31, false, false, -1},
			{15, 32, false, false, -1},
			{1, 33, true, false, -1}, // sync.
			{2, 34, false, false, -1},
			{3, 35, false, false, -1},
			{4, 36, false, false, -1},
			{5, 37, false, false, -1},
			{65533, 38, true, false, -1}, // sync.
			{65534, 39, false, false, -1},
			{65535, 40, false, false, -1},
			{0, 41, true, false, -1}, // sync.
			{1, 42, false, false, -1},
			{3, 0, false, true, -1}, // drop.
			{4, 44, false, false, -1},
			{5, 45, false, false, -1},
			{6, 46, false, false, -1},
			{7, 47, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("receive ordered numbers, sync, no drop, increase input", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{2, 2, false, false, -1},
			{80, 3, true, false, -1},
			{81, 4, false, false, -1},
			{82, 5, false, false, -1},
			{83, 6, false, false, -1},
			{84, 7, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("drop many inputs at the beginning (using uint16_t)", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{1, 1, false, false, -1},
			{2, 0, false, true, -1}, // drop.
			{3, 0, false, true, -1}, // drop.
			{4, 0, false, true, -1}, // drop.
			{5, 0, false, true, -1}, // drop.
			{6, 0, false, true, -1}, // drop.
			{7, 0, false, true, -1}, // drop.
			{8, 0, false, true, -1}, // drop.
			{9, 0, false, true, -1}, // drop.
			{120, 112, false, false, -1},
			{121, 113, false, false, -1},
			{122, 114, false, false, -1},
			{123, 115, false, false, -1},
			{124, 116, false, false, -1},
			{125, 117, false, false, -1},
			{126, 118, false, false, -1},
			{127, 119, false, false, -1},
			{128, 120, false, false, -1},
			{129, 121, false, false, -1},
			{130, 122, false, false, -1},
			{131, 123, false, false, -1},
			{132, 124, false, false, -1},
			{133, 125, false, false, -1},
			{134, 126, false, false, -1},
			{135, 127, false, false, -1},
			{136, 128, false, false, -1},
			{137, 129, false, false, -1},
			{138, 130, false, false, -1},
			{139, 131, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		seqManager2 := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
		validate(t, seqManager2, inputs)
	})

	t.Run("drop many inputs at the beginning (using uint8)", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint8]{
			{1, 1, false, false, -1},
			{2, 0, false, true, -1}, // drop.
			{3, 0, false, true, -1}, // drop.
			{4, 0, false, true, -1}, // drop.
			{5, 0, false, true, -1}, // drop.
			{6, 0, false, true, -1}, // drop.
			{7, 0, false, true, -1}, // drop.
			{8, 0, false, true, -1}, // drop.
			{9, 0, false, true, -1}, // drop.
			{120, 112, false, false, -1},
			{121, 113, false, false, -1},
			{122, 114, false, false, -1},
			{123, 115, false, false, -1},
			{124, 116, false, false, -1},
			{125, 117, false, false, -1},
			{126, 118, false, false, -1},
			{127, 119, false, false, -1},
			{128, 120, false, false, -1},
			{129, 121, false, false, -1},
			{130, 122, false, false, -1},
			{131, 123, false, false, -1},
			{132, 124, false, false, -1},
			{133, 125, false, false, -1},
			{134, 126, false, false, -1},
			{135, 127, false, false, -1},
			{136, 128, false, false, -1},
			{137, 129, false, false, -1},
			{138, 130, false, false, -1},
			{139, 131, false, false, -1},
		}

		seqManager := NewSeqManager[uint8]()
		validate(t, seqManager, inputs)
	})

	t.Run("receive mixed numbers, sync, drop in range 15", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 0, false, false, -1},
			{1, 1, false, false, -1},
			{2, 2, false, false, -1},
			{3, 3, false, false, -1},
			{7, 7, false, false, -1},
			{6, 0, false, true, -1}, // drop.
			{8, 8, false, false, -1},
			{10, 10, false, false, -1},
			{9, 9, false, false, -1},
			{11, 11, false, false, -1},
			{0, 12, true, false, -1}, // sync.
			{2, 14, false, false, -1},
			{3, 15, false, false, -1},
			{4, 16, false, false, -1},
			{5, 17, false, false, -1},
			{6, 18, false, false, -1},
			{7, 19, false, false, -1},
			{8, 20, false, false, -1},
			{9, 21, false, false, -1},
			{10, 22, false, false, -1},
			{9, 0, false, true, -1},   // drop.
			{61, 23, true, false, -1}, // sync.
			{62, 24, false, false, -1},
			{63, 25, false, false, -1},
			{64, 26, false, false, -1},
			{65, 27, false, false, -1},
			{11, 28, true, false, -1}, // sync.
			{12, 29, false, false, -1},
			{13, 30, false, false, -1},
			{14, 31, false, false, -1},
			{15, 32, false, false, -1},
			{1, 33, true, false, -1}, // sync.
			{2, 34, false, false, -1},
			{3, 35, false, false, -1},
			{4, 36, false, false, -1},
			{5, 37, false, false, -1},
			{32767, 38, true, false, -1}, // sync.
			{32768, 39, false, false, -1},
			{32769, 40, false, false, -1},
			{0, 41, true, false, -1}, // sync.
			{1, 42, false, false, -1},
			{3, 0, false, true, -1}, // drop.
			{4, 44, false, false, -1},
			{5, 45, false, false, -1},
			{6, 46, false, false, -1},
			{7, 47, false, false, -1},
		}

		seqManager := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
	})

	t.Run("drop many inputs at the beginning (using uint16_t with high values)", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{1, 1, false, false, -1},
			{2, 0, false, true, -1}, // drop.
			{3, 0, false, true, -1}, // drop.
			{4, 0, false, true, -1}, // drop.
			{5, 0, false, true, -1}, // drop.
			{6, 0, false, true, -1}, // drop.
			{7, 0, false, true, -1}, // drop.
			{8, 0, false, true, -1}, // drop.
			{9, 0, false, true, -1}, // drop.
			{32768, 32760, false, false, -1},
			{32769, 32761, false, false, -1},
			{32770, 32762, false, false, -1},
			{32771, 32763, false, false, -1},
			{32772, 32764, false, false, -1},
			{32773, 32765, false, false, -1},
			{32774, 32766, false, false, -1},
			{32775, 32767, false, false, -1},
			{32776, 32768, false, false, -1},
			{32777, 32769, false, false, -1},
			{32778, 32770, false, false, -1},
			{32779, 32771, false, false, -1},
			{32780, 32772, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		validate(t, seqManager, inputs)
	})

	t.Run("drop many inputs at the beginning (using uint16_t range 15 with high values)", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{1, 1, false, false, -1},
			{2, 0, false, true, -1}, // drop.
			{3, 0, false, true, -1}, // drop.
			{4, 0, false, true, -1}, // drop.
			{5, 0, false, true, -1}, // drop.
			{6, 0, false, true, -1}, // drop.
			{7, 0, false, true, -1}, // drop.
			{8, 0, false, true, -1}, // drop.
			{9, 0, false, true, -1}, // drop.
			{16384, 16376, false, false, -1},
			{16385, 16377, false, false, -1},
			{16386, 16378, false, false, -1},
			{16387, 16379, false, false, -1},
			{16388, 16380, false, false, -1},
			{16389, 16381, false, false, -1},
			{16390, 16382, false, false, -1},
			{16391, 16383, false, false, -1},
			{16392, 16384, false, false, -1},
			{16393, 16385, false, false, -1},
			{16394, 16386, false, false, -1},
			{16395, 16387, false, false, -1},
			{16396, 16388, false, false, -1},
		}

		seqManager := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
	})

	t.Run("sync and drop some input near max-value in a 15bit range", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{32762, 1, true, false, 32762},
			{32763, 2, false, false, 32763},
			{32764, 3, false, false, 32764},
			{32765, 0, false, true, 32765},
			{32766, 0, false, true, 32766},
			{32767, 4, false, false, 32767},
			{0, 5, false, false, 0},
			{1, 6, false, false, 1},
			{2, 7, false, false, 2},
			{3, 8, false, false, 3},
		}

		seqManager := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
	})

	t.Run("should update all values during multiple roll overs", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 1, true, false, 0},
		}

		for j := uint16(0); j < 3; j++ {
			for i := uint16(1); i < math.MaxUint16; i++ {
				output := i + 1
				inputs = append(inputs, &TestSeqManagerInput[uint16]{i, output, false, false, int64(i)})
			}
		}

		seqManager := NewSeqManager[uint16]()
		validate(t, seqManager, inputs)
	})

	t.Run("should update all values during multiple roll overs (15 bits range)", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{0, 1, true, false, 0},
		}

		for j := uint16(0); j < 3; j++ {
			for i := uint16(1); i < MaxNumberFor15Bits; i++ {
				output := i + 1
				inputs = append(inputs, &TestSeqManagerInput[uint16]{i, output, false, false, int64(i)})
			}
		}

		seqManager := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
	})

	t.Run("should produce same output for same old input before drop (15 bits range)", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{10, 1, true, false, -1}, // sync.
			{11, 2, false, false, -1},
			{12, 3, false, false, -1},
			{13, 4, false, false, -1},
			{14, 0, false, true, -1}, // drop.
			{15, 5, false, false, -1},
			{12, 3, false, false, -1},
		}

		seqManager := NewSeqManager[uint16](15)
		validate(t, seqManager, inputs)
	})

	t.Run("should properly clean previous cycle drops", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint8]{
			{1, 1, false, false, -1},
			{2, 0, false, true, -1}, // Drop.
			{3, 2, false, false, -1},
			{4, 3, false, false, -1},
			{5, 4, false, false, -1},
			{6, 5, false, false, -1},
			{7, 6, false, false, -1},
			{0, 7, false, false, -1},
			{1, 0, false, false, -1},
			{2, 1, false, false, -1},
			{3, 2, false, false, -1},
		}

		seqManager := NewSeqManager[uint8](3)
		validate(t, seqManager, inputs)
	})

	t.Run("dropped inputs to be removed going out of range, 1.", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{36964, 36964, false, false, -1},
			{25923, 0, false, true, -1}, // Drop.
			{25701, 25701, false, false, -1},
			{17170, 0, false, true, -1}, // Drop.
			{25923, 25923, false, false, -1},
			{4728, 0, false, true, -1}, // Drop.
			{17170, 17170, false, false, -1},
			{30738, 0, false, true, -1}, // Drop.
			{4728, 4728, false, false, -1},
			{4806, 0, false, true, -1}, // Drop.
			{30738, 30738, false, false, -1},
			{50886, 0, false, true, -1},    // Drop.
			{4806, 4805, false, false, -1}, // Previously dropped.
			{50774, 0, false, true, -1},    // Drop.
			{50886, 0, false, false, -1},   // Previously dropped.
			{22136, 0, false, true, -1},    // Drop.
			{50774, 50773, false, false, -1},
			{30910, 0, false, true, -1},  // Drop.
			{22136, 0, false, false, -1}, // Previously dropped.
			{48862, 0, false, true, -1},  // Drop.
			{30910, 30909, false, false, -1},
			{56832, 0, false, true, -1}, // Drop.
			{48862, 48861, false, false, -1},
			{2, 0, false, true, -1},      // Drop.
			{56832, 0, false, false, -1}, // Previously dropped.
			{530, 0, false, true, -1},    // Drop.
			{2, 0, false, false, -1},     // Previously dropped.
		}

		seqManager := NewSeqManager[uint16]()
		validate(t, seqManager, inputs)
	})

	t.Run("dropped inputs to be removed go out of range, 2.", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{36960, 36960, false, false, -1},
			{3328, 0, false, true, -1}, // Drop.
			{24589, 24588, false, false, -1},
			{120, 0, false, true, -1},   // Drop.
			{3328, 0, false, false, -1}, // Previously dropped.
			{30848, 0, false, true, -1}, // Drop.
			{120, 120, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		validate(t, seqManager, inputs)
	})

	t.Run("dropped inputs to be removed go out of range, 3.", func(t *testing.T) {
		inputs := []*TestSeqManagerInput[uint16]{
			{36964, 36964, false, false, -1},
			{65396, 0, false, true, -1}, // Drop.
			{25855, 25854, false, false, -1},
			{29793, 0, false, true, -1},  // Drop.
			{65396, 0, false, false, -1}, // Previously dropped.
			{25087, 0, false, true, -1},  // Drop.
			{29793, 0, false, false, -1}, // Previously dropped.
			{65535, 0, false, true, -1},  // Drop.
			{25087, 25086, false, false, -1},
		}

		seqManager := NewSeqManager[uint16]()
		validate(t, seqManager, inputs)
	})
}

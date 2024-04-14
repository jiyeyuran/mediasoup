package rtc

import (
	"github.com/google/btree"
	"github.com/jiyeyuran/mediasoup/internal/util"
	"golang.org/x/exp/constraints"
)

func IsSeqLowerThan[T constraints.Unsigned](lhs, rhs T, max ...T) bool {
	var maxValue T
	if len(max) > 0 {
		maxValue = max[0]
	} else {
		maxValue = util.MaxOf[T]()
	}

	return ((rhs > lhs) && (rhs-lhs <= maxValue/2)) ||
		((lhs > rhs) && (lhs-rhs > maxValue/2))
}

func IsSeqHigherThan[T constraints.Unsigned](lhs, rhs T, max ...T) bool {
	var maxValue T
	if len(max) > 0 {
		maxValue = max[0]
	} else {
		maxValue = util.MaxOf[T]()
	}

	return ((lhs > rhs) && (lhs-rhs <= maxValue/2)) ||
		((rhs > lhs) && (rhs-lhs > maxValue/2))
}

type SeqManager[T constraints.Unsigned] struct {
	maxValue                  T
	started                   bool
	base, maxOutput, maxInput T
	dropped                   *btree.BTreeG[T]
}

// NewSeqManager n is the max number of bits used in T.
func NewSeqManager[T constraints.Unsigned](n ...int) *SeqManager[T] {
	var maxValue T
	if len(n) > 0 {
		maxValue = 1<<n[0] - 1
	} else {
		maxValue = util.MaxOf[T]()
	}

	return &SeqManager[T]{
		maxValue: maxValue,
		dropped: btree.NewG[T](2, func(lhs, rhs T) bool {
			return IsSeqLowerThan(lhs, rhs, maxValue)
		}),
	}
}

func (s *SeqManager[T]) Sync(input T) {
	// Update base.
	s.base = (s.maxOutput - input) & s.maxValue

	// Update maxInput.
	s.maxInput = input

	// Clear dropped set.
	s.dropped.Clear(false)
}

func (s *SeqManager[T]) Drop(input T) {
	if s.isSeqHigherThan(input, s.maxInput) {
		s.maxInput = input

		// Insert input.
		s.dropped.ReplaceOrInsert(input)

		s.clearDropped()
	}
}

func (s *SeqManager[T]) Input(input T) (output T, ok bool) {
	base := s.base

	// No dropped inputs to consider.
	if s.dropped.Len() == 0 {
		goto done
	} else { // Dropped inputs present, cleanup and update base.
		// Set 'maxInput' here if needed before calling clearDropped().
		if s.started && s.isSeqHigherThan(input, s.maxInput) {
			s.maxInput = input
		}

		s.clearDropped()

		base = s.base
	}

	// No dropped inputs to consider after cleanup.
	if s.dropped.Len() == 0 {
		goto done
	} else if _, found := s.dropped.Get(input); found { // This input was dropped.
		// trying to send a dropped input
		return 0, false
	} else { // There are dropped inputs, calculate 'base' for this input.
		droppedCount := T(s.dropped.Len())

		// Decrease dropped input which is higher than or equal 'input'.
		s.dropped.Descend(func(value T) bool {
			if s.isSeqHigherThan(value, input) || value == input {
				droppedCount--
				return true
			}
			return false
		})
		base = (s.base - droppedCount) & s.maxValue
	}

done:
	output = (input + base) & s.maxValue

	if !s.started {
		s.started = true
		s.maxInput = input
		s.maxOutput = output
	} else {
		// New input is higher than the maximum seen.
		if s.isSeqHigherThan(input, s.maxInput) {
			s.maxInput = input
		}

		// New output is higher than the maximum seen.
		if s.isSeqHigherThan(output, s.maxOutput) {
			s.maxOutput = output
		}
	}

	return output, true
}

func (s *SeqManager[T]) GetMaxInput() T {
	return s.maxInput
}

func (s *SeqManager[T]) GetMaxOutput() T {
	return s.maxOutput
}

// clearDropped delete droped inputs greater than maxInput, which belong to a previous cycle.
func (s *SeqManager[T]) clearDropped() {
	previousDroppedSize := s.dropped.Len()

	// Cleanup dropped values.
	if previousDroppedSize == 0 {
		return
	}

	var del []T

	s.dropped.Ascend(func(value T) bool {
		if s.isSeqHigherThan(value, s.maxInput) {
			del = append(del, value)
			return true
		}
		return false
	})

	for _, v := range del {
		s.dropped.Delete(v)
	}

	// Adapt base.
	s.base = (s.base - T(previousDroppedSize-s.dropped.Len())) & s.maxValue
}

func (s *SeqManager[T]) isSeqLowerThan(lhs, rhs T) bool {
	return IsSeqLowerThan(lhs, rhs, s.maxValue)
}

func (s *SeqManager[T]) isSeqHigherThan(lhs, rhs T) bool {
	return IsSeqHigherThan(lhs, rhs, s.maxValue)
}

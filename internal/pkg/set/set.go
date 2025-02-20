package set

import (
	"github.com/zhangyunhao116/skipset"
)

type Set[T any] struct {
	*skipset.FuncSet[T]
	less func(a, b T) bool
}

func NewFunc[T any](less func(a, b T) bool) *Set[T] {
	return &Set[T]{
		FuncSet: skipset.NewFunc(less),
		less:    less,
	}
}

// DeleteLessThan delete items in [first, last)
func (s *Set[T]) DeleteLessThan(last T) {
	s.Range(func(value T) bool {
		if s.less(value, last) {
			s.Remove(value)
			return true
		} else {
			return false
		}
	})
}

func (s *Set[T]) Clear() {
	s.FuncSet = skipset.NewFunc(s.less)
}

func (s Set[T]) First() (val T, ok bool) {
	s.Range(func(value T) bool {
		val = value
		ok = true
		return false
	})
	return val, ok
}

func (s Set[T]) Last() (val T, ok bool) {
	s.Range(func(value T) bool {
		val = value
		ok = true
		return true
	})
	return val, ok
}

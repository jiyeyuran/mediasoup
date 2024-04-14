package util

import (
	"unsafe"

	"golang.org/x/exp/constraints"
)

// MaxOf returns maximum value of type T.
func MaxOf[T constraints.Integer]() T {
	if ^T(0) > 0 {
		return ^T(0)
	}
	var v T
	bits := 8 * unsafe.Sizeof(v)
	return 1<<(bits-1) - 1
}

// MinOf returns minimum value of type T.
func MinOf[T constraints.Integer]() T {
	if ^T(0) > 0 {
		return 0
	}
	var v T
	bits := 8 * unsafe.Sizeof(v)
	return (^v) << (bits - 1)
}

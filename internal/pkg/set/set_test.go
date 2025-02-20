package set

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFunc(t *testing.T) {
	s := NewFunc(func(a, b int) bool { return a < b })
	require.NotNil(t, s, "NewFunc returned nil")
}

func TestDeleteLessThan(t *testing.T) {
	less := func(a, b int) bool { return a < b }

	t.Run("Empty set", func(t *testing.T) {
		s1 := NewFunc(less)
		s1.DeleteLessThan(5)
		require.Equal(t, 0, s1.Len(), "DeleteLessThan on empty set should not change the set")
	})

	t.Run("Single element", func(t *testing.T) {
		s2 := NewFunc(less)
		s2.Add(3)
		s2.DeleteLessThan(3)
		require.Equal(t, 1, s2.Len(), "DeleteLessThan failed on single element set")
		require.True(t, s2.Contains(3), "DeleteLessThan failed on single element set")

		s2.DeleteLessThan(5)
		require.Equal(t, 0, s2.Len(), "DeleteLessThan failed on single element set")

		s2.Add(3)
		s2.DeleteLessThan(4)
		require.Equal(t, 0, s2.Len(), "DeleteLessThan failed on single element set")
	})

	t.Run("Multiple elements", func(t *testing.T) {
		s3 := NewFunc(less)
		s3.Add(1)
		s3.Add(2)
		s3.Add(3)
		s3.Add(4)
		s3.Add(5)
		s3.DeleteLessThan(3)
		require.Equal(t, 3, s3.Len(), "DeleteLessThan failed on multiple element set")
		require.True(t, s3.Contains(3), "DeleteLessThan failed on multiple element set")
		require.True(t, s3.Contains(4), "DeleteLessThan failed on multiple element set")
		require.True(t, s3.Contains(5), "DeleteLessThan failed on multiple element set")

		s3.DeleteLessThan(6)
		require.Equal(t, 0, s3.Len(), "DeleteLessThan failed on multiple element set")
	})
}

func TestClear(t *testing.T) {
	less := func(a, b int) bool { return a < b }

	t.Run("Empty set", func(t *testing.T) {
		s1 := NewFunc(less)
		s1.Clear()
		require.Equal(t, 0, s1.Len(), "Clear on empty set failed")
	})

	t.Run("Multiple elements", func(t *testing.T) {
		s2 := NewFunc(less)
		s2.Add(1)
		s2.Add(2)
		s2.Add(3)
		s2.Clear()
		require.Equal(t, 0, s2.Len(), "Clear on multiple element set failed")
	})
}

func TestFirst(t *testing.T) {
	less := func(a, b int) bool { return a < b }

	t.Run("Empty set", func(t *testing.T) {
		s1 := NewFunc(less)
		_, ok := s1.First()
		require.False(t, ok, "First on empty set should return false")
	})

	t.Run("Single element", func(t *testing.T) {
		s2 := NewFunc(less)
		s2.Add(5)
		val, ok := s2.First()
		require.True(t, ok, "First on single element set failed")
		require.Equal(t, 5, val, "First on single element set failed")
	})

	t.Run("Multiple elements", func(t *testing.T) {
		s3 := NewFunc(less)
		s3.Add(3)
		s3.Add(1)
		s3.Add(5)
		val, ok := s3.First()
		require.True(t, ok, "First on multiple element set failed")
		require.Equal(t, 1, val, "First on multiple element set failed")
	})
}

func TestLast(t *testing.T) {
	less := func(a, b int) bool { return a < b }

	t.Run("Empty set", func(t *testing.T) {
		s1 := NewFunc(less)
		_, ok := s1.Last()
		require.False(t, ok, "Last on empty set should return false")
	})

	t.Run("Single element", func(t *testing.T) {
		s2 := NewFunc(less)
		s2.Add(5)
		val, ok := s2.Last()
		require.True(t, ok, "Last on single element set failed")
		require.Equal(t, 5, val, "Last on single element set failed")
	})

	t.Run("Multiple elements", func(t *testing.T) {
		s3 := NewFunc(less)
		s3.Add(3)
		s3.Add(1)
		s3.Add(5)
		val, ok := s3.Last()
		require.True(t, ok, "Last on multiple element set failed")
		require.Equal(t, 5, val, "Last on multiple element set failed")
	})
}

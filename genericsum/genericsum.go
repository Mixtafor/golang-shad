//go:build !solution

package genericsum

import (
	"math/cmplx"
	"slices"
	"sync"

	"golang.org/x/exp/constraints"
)

func Min[T constraints.Ordered](a, b T) T {
	if a <= b {
		return a
	}
	return b
}

func SortSlice[T constraints.Ordered, E ~[]T](a E) {
	slices.Sort(a)
}

func MapsEqual[K, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 {
		return true
	}

	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

func SliceContains[T comparable, E ~[]T](s E, v T) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

func MergeChans[T any](chs ...<-chan T) <-chan T {
	wg := sync.WaitGroup{}
	totalCap := 0
	for _, ch := range chs {
		totalCap += cap(ch)
	}

	mainCh := make(chan T, totalCap)

	for _, ch := range chs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for v := range ch {
				mainCh <- v
			}
		}()
	}

	// close mainCh
	go func() {
		wg.Wait()
		close(mainCh)
	}()

	return mainCh
}

type Numeric interface {
	constraints.Integer | constraints.Float | constraints.Complex
}

func IsHermitianMatrix[T Numeric](m [][]T) bool {
	if len(m) == 0 {
		return true
	}

	for _, row := range m {
		if len(row) != len(m) {
			return false
		}
	}

	var matrx any = m

	switch mTyped := matrx.(type) {
	case [][]complex64:
		for i := range len(m) {
			if imag(mTyped[i][i]) != 0 {
				return false
			}

			for j := i + 1; j < len(m); j++ {
				if cmplx.Conj(complex128(mTyped[i][j])) != complex128(mTyped[j][i]) {
					return false
				}
			}
		}
	case [][]complex128:
		for i := range len(m) {
			if imag(mTyped[i][i]) != 0 {
				return false
			}

			for j := i + 1; j < len(m); j++ {
				if cmplx.Conj(complex128(mTyped[i][j])) != complex128(mTyped[j][i]) {
					return false
				}
			}
		}
	default:
		for i := range len(m) {
			for j := i + 1; j < len(m); j++ {
				if m[i][j] != m[j][i] {
					return false
				}
			}
		}
	}

	return true
}

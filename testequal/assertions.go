//go:build !solution

package testequal

import (
	"fmt"
	"maps"
	"slices"
)

// AssertEqual checks that expected and actual are equal.
//
// Marks caller function as having failed but continues execution.
//
// Returns true iff arguments are equal.
func createFailMsg(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	format, ok := msgAndArgs[0].(string)
	if !ok {
		return ""
	}
	return fmt.Sprintf(format, msgAndArgs[1:]...)
}

func AssertEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	t.Helper()

	switch v := expected.(type) {
	case string:
		casted, ok := actual.(string)
		if !ok || v != casted {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case []int:
		casted, ok := actual.([]int)
		if !ok || (v == nil) != (casted == nil) || !slices.Equal(v, casted) {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case []byte:
		casted, ok := actual.([]byte)
		if !ok || (v == nil) != (casted == nil) || !slices.Equal(v, casted) {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case map[string]string:
		casted, ok := actual.(map[string]string)
		if !ok || (v == nil) != (casted == nil) || !maps.Equal(v, casted) {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr:
		if expected != actual {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true

	default:
		return false
	}
}

// AssertNotEqual checks that expected and actual are not equal.
//
// Marks caller function as having failed but continues execution.
//
// Returns true iff arguments are not equal.
func AssertNotEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	t.Helper()

	switch v := expected.(type) {
	case string:
		casted, ok := actual.(string)
		if ok && v == casted {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case []int:
		casted, ok := actual.([]int)
		if ok && (v == nil) == (casted == nil) && slices.Equal(v, casted) {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case []byte:
		casted, ok := actual.([]byte)
		if ok && (v == nil) == (casted == nil) && slices.Equal(v, casted) {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case map[string]string:
		casted, ok := actual.(map[string]string)
		if ok && (v == nil) == (casted == nil) && maps.Equal(v, casted) {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr:
		if expected == actual {
			t.Errorf(createFailMsg(msgAndArgs...))
			return false
		}
		return true

	default:
		return true
	}
}

// RequireEqual does the same as AssertEqual but fails caller test immediately.
func RequireEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !AssertEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

// RequireNotEqual does the same as AssertNotEqual but fails caller test immediately.
func RequireNotEqual(t T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !AssertNotEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

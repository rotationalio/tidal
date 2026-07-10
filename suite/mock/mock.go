package mock

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Mock is the base type for all mocks.
type Mock struct {
	calls map[string]int
}

//============================================================================
// Helper Methods
//============================================================================

func (m *Mock) Reset() {
	m.calls = nil
}

func (m *Mock) increment(method string) {
	if m.calls == nil {
		m.calls = make(map[string]int)
	}
	m.calls[method]++
}

//============================================================================
// Assertions
//============================================================================

func (m *Mock) Calls(method string) int {
	n := 0
	if method == "Fields" || method == "Params" {
		mu.Lock()
		defer mu.Unlock()
		if globalCalls != nil {
			n = globalCalls[method]
		}
	}

	if m.calls == nil {
		return n
	}
	return n + m.calls[method]
}

func (m *Mock) AssertCalled(t *testing.T, method string) {
	require.Greater(t, m.Calls(method), 0, "expected %s to be called at least once", method)
}

func (m *Mock) AssertNotCalled(t *testing.T, method string) {
	require.Equal(t, 0, m.Calls(method), "expected %s to not be called", method)
}

func (m *Mock) AssertCalls(t *testing.T, method string, count int) {
	require.Equal(t, count, m.Calls(method), "expected %s to be called %d times", method, count)
}

func (m *Mock) AssertCalledOnce(t *testing.T, method string) {
	require.Equal(t, 1, m.Calls(method), "expected %s to be called exactly once", method)
}

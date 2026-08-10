package testing

import testing_original "testing"

func TestContextReturnsContext(t *testing_original.T) {
	simulated := &T{}
	if simulated.Context() == nil {
		t.Fatal("Context returned nil")
	}
}

package examples_test

import (
	"math/rand"
	"testing"

	"github.com/temporalio/gomad"
)

func TestGomad(t *testing.T) {
	t.Logf("Are we in the Matrix? %v", gomad.IsSim())
	t.Logf("Random: %d", rand.Int())
}

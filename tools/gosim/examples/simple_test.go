package examples_test

import (
	"math/rand"
	"testing"

	"github.com/jellevandenhooff/gosim"
)

func TestGosim(t *testing.T) {
	t.Logf("Are we in the Matrix? %v", gosim.IsSim())
	t.Logf("Random: %d", rand.Int())
}

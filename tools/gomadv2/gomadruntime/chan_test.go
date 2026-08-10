package gomadruntime_test

import (
	"testing"

	"github.com/temporalio/gomad/gomadruntime"
)

func TestNewChanAcceptsIntegerCapacity(t *testing.T) {
	var size int32 = 2
	ch := gomadruntime.NewChan[int](size)
	if got := ch.Cap(); got != int(size) {
		t.Fatalf("capacity = %d, want %d", got, size)
	}
}

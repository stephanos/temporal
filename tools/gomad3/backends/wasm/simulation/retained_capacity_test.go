package simulation

import (
	"errors"
	"testing"
)

func TestRetentionCapacityPreflightUsesEncodedLowerBound(t *testing.T) {
	if err := retainedCapacityLowerBound(0, []uint64{240271906, 13713712}); err != nil {
		t.Fatal(err)
	}
	var capacity *RetentionCapacityError
	if err := retainedCapacityLowerBound(0, []uint64{240271906, 240271906, 13713712}); !errors.As(err, &capacity) || capacity.Required != 640725088 || capacity.Maximum != 512<<20 {
		t.Fatalf("distinct large modules were not rejected before allocation: %+v %v", capacity, err)
	}
	if err := retainedCapacityLowerBound(512<<20, nil); !errors.Is(err, ErrRetentionCapacity) {
		t.Fatalf("record base64 growth was not admitted: %v", err)
	}
	if err := retainedCapacityLowerBound(0, []uint64{^uint64(0)}); !errors.Is(err, ErrRetentionCapacity) {
		t.Fatalf("input size overflow bypassed capacity: %v", err)
	}
}

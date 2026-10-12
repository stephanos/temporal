package simulation

import (
	"errors"
	"testing"
)

func TestRetentionCapacityPreflightUsesEncodedLowerBound(t *testing.T) {
	if err := retainedCapacityLowerBound(66 << 20); err != nil {
		t.Fatal(err)
	}
	var capacity *RetentionCapacityError
	if err := retainedCapacityLowerBound(400 << 20); !errors.As(err, &capacity) || capacity.Required != 4*((400<<20+2)/3) || capacity.Maximum != 512<<20 {
		t.Fatalf("record base64 growth was not rejected before encoding: %+v %v", capacity, err)
	}
	if err := retainedCapacityLowerBound(^uint64(0)); !errors.Is(err, ErrRetentionCapacity) {
		t.Fatalf("record size overflow bypassed capacity: %v", err)
	}
}

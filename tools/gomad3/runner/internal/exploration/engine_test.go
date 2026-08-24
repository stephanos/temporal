package exploration

import (
	"errors"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestEngineOwnsDeterministicRoundAndIdentitySetMechanics(t *testing.T) {
	queue := []int{1, 2, 3}
	round, ok := NextRound(queue, 2, false, func(value int) int { return value })
	if !ok || len(round) != 2 || round[0] != 1 || round[1] != 2 {
		t.Fatalf("NextRound() = %#v, %t", round, ok)
	}
	if _, ok := NextRound(queue, 2, true, func(value int) int { return value }); ok {
		t.Fatal("NextRound() returned a stopped round")
	}

	one := record.HashBytes([]byte("one"))
	two := record.HashBytes([]byte("two"))
	identities := InsertIdentity(nil, two)
	identities = InsertIdentity(identities, one)
	identities = InsertIdentity(identities, two)
	if len(identities) != 2 || !ContainsIdentity(identities, one) || !ContainsIdentity(identities, two) {
		t.Fatalf("identity set = %#v", identities)
	}

	total, err := SumBytes([]int{1, 2, 3}, func(value int) (uint64, error) { return uint64(value), nil })
	if err != nil || total != 6 {
		t.Fatalf("SumBytes() = %d, %v", total, err)
	}
	_, err = SumBytes([]int{1}, func(int) (uint64, error) { return 0, errors.New("size") })
	if err == nil {
		t.Fatal("SumBytes() accepted a sizing error")
	}
}

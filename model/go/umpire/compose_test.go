package umpire_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/umpire"
)

func TestACompositionStartsInEveryAdmittedStart(t *testing.T) {
	either := newDoor("door").Starts(door{Phase: closed}, door{Phase: closed, Oiled: true})
	tb := tableOf(t, umpire.Compose[house]("test.door", "house").
		Member("door", either).Member("key", keyholder).Sync("lock", "door.lock", "key.useKey"))
	require.Equal(t, []string{"closed-false_holding", "closed-true_holding"}, tb.Starts)
	require.Contains(t, tb.Reachable, "open-true_holding", "an oiled door opens from its own start")

	single := tableOf(t, umpire.Compose[house]("test.door", "house").
		Member("door", newDoor("door")).Member("key", keyholder).Sync("lock", "door.lock", "key.useKey"))
	require.Equal(t, []string{"closed-false_holding"}, single.Starts)
	require.NotContains(t, single.Reachable, "open-true_holding", "no step oils the door")
}

func TestASyncNamingAnActionItsMemberLacksIsRejected(t *testing.T) {
	_, err := umpire.Compose[house]("test.door", "bad").
		Member("door", newDoor("door")).Member("key", keyholder).Sync("fly", "door.fly", "key.useKey").Table()
	require.EqualError(t, err, "compose-bad: sync fly names door.fly, and door has no action fly")
}

// An opaque keyholder that may use or lose the key, and a detailed one that also wears it down.

var keyIsOpaque = umpire.Assumption{Name: "keyIsOpaque"}

func opaqueKey() *umpire.Machine[keyState, doorOutcome, keyFact] {
	return umpire.NewMachine[keyState, doorOutcome, keyFact]("test.door", "opaqueKey").
		Starts(keyState{holding}).
		Assumes(keyIsOpaque).
		Step0(useKey, func(k keyState) []keyStep {
			if k.Phase != holding {
				return nil
			}
			return []keyStep{{Outcome: ok, State: k}}
		}).
		Step0(loseKey, func(k keyState) []keyStep {
			if k.Phase != holding {
				return nil
			}
			return []keyStep{{Outcome: ok, State: keyState{lost}}}
		})
}

type wornKey struct {
	Phase keyPhase
	Worn  bool
}

type wornStep = umpire.Step[wornKey, doorOutcome, keyFact]

var findKey = umpire.NewAction0("findKey", "person")

func detailedKey(name string, finds bool) *umpire.Machine[wornKey, doorOutcome, keyFact] {
	m := umpire.NewMachine[wornKey, doorOutcome, keyFact]("test.door", name).
		Starts(wornKey{Phase: holding}).
		Step0(useKey, func(k wornKey) []wornStep {
			if k.Phase != holding {
				return nil
			}
			return []wornStep{{Outcome: ok, State: wornKey{Phase: holding, Worn: true}}}
		}).
		Step0(loseKey, func(k wornKey) []wornStep {
			if k.Phase != holding {
				return nil
			}
			return []wornStep{{Outcome: ok, State: wornKey{Phase: lost, Worn: k.Worn}}}
		})
	if finds {
		m.Step0(findKey, func(k wornKey) []wornStep {
			if k.Phase != lost {
				return nil
			}
			return []wornStep{{Outcome: ok, State: wornKey{Phase: holding, Worn: k.Worn}}}
		})
	}
	return m
}

func keyOfWorn(k wornKey) keyState { return keyState{k.Phase} }

type wornHouse struct {
	Door door    `umpire:"door"`
	Key  wornKey `umpire:"key"`
}

var doorIsOiled = umpire.Assumption{Name: "doorIsOiled"}

func TestAReplacingMemberStandsInForTheOpaqueProvider(t *testing.T) {
	opaque := opaqueKey()
	detailed := detailedKey("detailedKey", false).Refines(opaque, keyOfWorn)
	c := umpire.Compose[wornHouse]("test.door", "house").
		Member("door", newDoor("door").Assumes(doorIsOiled)).Member("key", detailed).Replaces("key", opaque).
		Sync("lock", "door.lock", "key.useKey")
	tb := tableOf(t, c)
	require.Equal(t, []umpire.Assumption{doorIsOiled}, tb.Assumptions, "the check relies on none of the opaque key's")
	require.Contains(t, tb.Reachable, "locked-false_holding-true")

	withOpaque := tableOf(t, umpire.Compose[house]("test.door", "house").
		Member("door", newDoor("door").Assumes(doorIsOiled)).Member("key", opaque).
		Sync("lock", "door.lock", "key.useKey"))
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, withOpaque.Assumptions)
}

func TestAReplacementDischargesOnlyTheReplacingMembersAssumptions(t *testing.T) {
	opaque := opaqueKey()
	inherits := detailedKey("detailedKey", false).Assumes(keyIsOpaque).Refines(opaque, keyOfWorn)
	tb := tableOf(t, umpire.Compose[wornHouse]("test.door", "house").
		Member("door", newDoor("door").Assumes(doorIsOiled)).Member("key", inherits).Replaces("key", opaque).
		Sync("lock", "door.lock", "key.useKey"))
	require.Equal(t, []umpire.Assumption{doorIsOiled}, tb.Assumptions, "the replacing member no longer relies on it")

	opaque = opaqueKey()
	inherits = detailedKey("detailedKey", false).Assumes(keyIsOpaque).Refines(opaque, keyOfWorn)
	tb = tableOf(t, umpire.Compose[wornHouse]("test.door", "house").
		Member("door", newDoor("door").Assumes(doorIsOiled, keyIsOpaque)).Member("key", inherits).
		Replaces("key", opaque).Sync("lock", "door.lock", "key.useKey"))
	require.Equal(t, []umpire.Assumption{doorIsOiled, keyIsOpaque}, tb.Assumptions,
		"the door relies on the opaque key's assumption on its own")
}

func TestAMembersFairnessNamesItsComposedClasses(t *testing.T) {
	fairDoor := umpire.Assumption{Name: "doorIsFair", Fair: []string{"push", "lock"}}
	c := umpire.Compose[house]("test.door", "house").
		Member("door", newDoor("door").Assumes(fairDoor)).Member("key", keyholder).
		Sync("lock", "door.lock", "key.useKey")
	tb := tableOf(t, c)
	require.Equal(t, []umpire.Assumption{{Name: "doorIsFair", Fair: []string{"door_push", "lock"}}}, tb.Assumptions)

	shuts := umpire.NewProgress("shuts", func(h house) bool { return h.Door.Phase == open },
		func(h house) bool { return h.Door.Phase == closed }, 1)
	a, err := umpire.CheckProgress(tb, shuts, wide)
	require.NoError(t, err)
	require.Equal(t, []string{"doorIsFair"}, a.Assumptions)
	requireVerdict(t, umpire.VerifiedWithinLimits, a.Cycle)

	ghost := umpire.Assumption{Name: "ghostIsFair", Fair: []string{"ghost"}}
	_, err = umpire.Compose[house]("test.door", "haunted").
		Member("door", newDoor("door").Assumes(ghost)).Member("key", keyholder).Table()
	require.EqualError(t, err, "compose-haunted: the assumption ghostIsFair of door makes ghost fair, which is no action of door")
}

func TestAViolatingProviderFailsItsReplacement(t *testing.T) {
	opaque := opaqueKey()
	finder := detailedKey("finderKey", true).Refines(opaque, keyOfWorn)
	_, err := umpire.Compose[wornHouse]("test.door", "house").
		Member("door", newDoor("door")).Member("key", finder).Replaces("key", opaque).
		Sync("lock", "door.lock", "key.useKey").Table()
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementUnmatched, re.Kind)
	require.ErrorContains(t, err, "finderKey refines opaqueKey: the row 'lost-false-findKey'")
	require.Equal(t, []string{"loseKey", "findKey"}, actionsOf(re.Witness))
	requireReplaysFromAStart(t, finder, re.Witness)
}

func TestAReplacementAccountsForEveryOpaqueStart(t *testing.T) {
	opaque := opaqueKey().Starts(keyState{holding}, keyState{lost})
	detailed := detailedKey("detailedKey", false).Refines(opaque, keyOfWorn)
	_, err := detailed.Refinement()
	require.NoError(t, err, "on its own, the refinement may start in fewer states")
	_, err = umpire.Compose[wornHouse]("test.door", "house").
		Member("door", newDoor("door")).Member("key", detailed).Replaces("key", opaque).
		Sync("lock", "door.lock", "key.useKey").Table()
	require.EqualError(t, err, "detailedKey refines opaqueKey: opaqueKey starts at 'lost', which no start of detailedKey reads as")
	require.Equal(t, umpire.RefinementInitial, refinementError(t, err).Kind)
}

func TestAReplacementIsDeclaredAgainstARefinementOfAMember(t *testing.T) {
	opaque := opaqueKey()
	_, err := umpire.Compose[wornHouse]("test.door", "house").
		Member("door", newDoor("door")).Member("key", detailedKey("unrelated", false)).Replaces("key", opaque).Table()
	require.EqualError(t, err, "compose-house: the member key replaces opaqueKey, and unrelated does not refine it")

	_, err = umpire.Compose[wornHouse]("test.door", "house").
		Member("door", newDoor("door")).Replaces("key", opaque).Table()
	require.EqualError(t, err, "compose-house: the composition replaces opaqueKey at key, and has no member key")
}

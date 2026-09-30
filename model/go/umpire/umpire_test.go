package umpire_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/umpire"
)

// A toy door: closed, open or locked, with a knob that turns by one of two hands.

type doorPhase string

const (
	closed doorPhase = "closed"
	open   doorPhase = "open"
	locked doorPhase = "locked"
)

func (doorPhase) Values() []doorPhase { return []doorPhase{closed, open, locked} }

type door struct {
	Phase doorPhase
	Oiled bool
}

type doorOutcome string

const ok doorOutcome = "ok"

func (doorOutcome) Values() []doorOutcome { return []doorOutcome{ok} }

type doorFact interface{ isDoorFact() }

type (
	opened   struct{}
	creaked  struct{ Loud bool }
	unlocked struct{}
)

func (opened) isDoorFact()   {}
func (creaked) isDoorFact()  {}
func (unlocked) isDoorFact() {}

var _ = umpire.Sum[doorFact](opened{}, creaked{}, unlocked{})

type hand interface{ isHand() }

type (
	left  struct{}
	right struct{ Strong bool }
)

func (left) isHand()  {}
func (right) isHand() {}

var _ = umpire.Sum[hand](left{}, right{})

type doorStep = umpire.Step[door, doorOutcome, doorFact]

var (
	turn = umpire.NewAction1[hand]("turn", "person", "hand")
	push = umpire.NewAction0("push", "person")
	lock = umpire.NewAction0("lock", "person")
)

func turnStep(d door, h hand) []doorStep {
	if d.Phase != closed {
		return nil
	}
	if r, isRight := h.(right); isRight && r.Strong {
		d.Phase = open
		return []doorStep{{Outcome: ok, State: d, Facts: []doorFact{opened{}, creaked{Loud: !d.Oiled}}}}
	}
	return nil
}

func pushStep(d door) []doorStep {
	if d.Phase != open {
		return nil
	}
	d.Phase = closed
	return []doorStep{{Outcome: ok, State: d}}
}

func lockStep(d door) []doorStep {
	if d.Phase != closed {
		return nil
	}
	d.Phase = locked
	return []doorStep{{Outcome: ok, State: d}}
}

func newDoor(name string) *umpire.Machine[door, doorOutcome, doorFact] {
	return umpire.NewMachine[door, doorOutcome, doorFact]("test.door", name).
		Starts(door{Phase: closed}).
		Ends(func(d door) bool { return d.Phase == locked }).
		Evidence("opened", "opened").Evidence("creaked", "creaked").
		Step1(turn, turnStep).Step0(push, pushStep).Step0(lock, lockStep)
}

func tableOf(t *testing.T, m umpire.Model) *umpire.Table {
	t.Helper()
	tb, err := m.Table()
	require.NoError(t, err)
	return tb
}

func TestDomainOrderIsDeclarationOrderWithTheLastFieldFastest(t *testing.T) {
	states, err := umpire.DomainOf[door]()
	require.NoError(t, err)
	var keys []string
	for _, s := range states {
		keys = append(keys, umpire.KeyOf(s))
	}
	require.Equal(t, []string{"closed-false", "closed-true", "open-false", "open-true", "locked-false", "locked-true"}, keys)

	hands, err := umpire.DomainOf[hand]()
	require.NoError(t, err)
	var handKeys []string
	for _, h := range hands {
		handKeys = append(handKeys, umpire.KeyOf(h))
	}
	require.Equal(t, []string{"left", "right-false", "right-true"}, handKeys)
}

func TestSumOfANonInterfaceIsReportedWhenEnumerated(t *testing.T) {
	type notAnInterface struct{ X bool }
	_ = umpire.Sum[notAnInterface]()
	_, err := umpire.DomainOf[notAnInterface]()
	require.ErrorContains(t, err, "is not an interface")
}

func TestUnenumerableTypeIsRejected(t *testing.T) {
	type bad struct{ N float64 }
	_, err := umpire.DomainOf[bad]()
	require.ErrorContains(t, err, "is not finite")
}

func TestTableOrdersActionsByKeyAndRowsStatesMajor(t *testing.T) {
	tb := tableOf(t, newDoor("door"))
	require.Equal(t, []string{"lock", "push", "turn-left", "turn-right-false", "turn-right-true"}, tb.Actions)
	var rows []string
	for _, r := range tb.Rows {
		rows = append(rows, r.Key)
	}
	require.Equal(t, []string{"closed-false-lock", "closed-false-turn-right-true", "closed-true-lock",
		"closed-true-turn-right-true", "open-false-push", "open-true-push"}, rows)
	require.Equal(t, []string{"closed-false", "locked-false", "open-false"}, tb.Reachable)
	require.Equal(t, []string{"locked-false", "locked-true"}, tb.Ends)
	require.Equal(t, []string{"opened", "creaked-false", "creaked-true", "unlocked"}, tb.Facts)
	require.Equal(t, "test.door.state.door.closed-false", tb.IDs().States[0])
}

func TestAStepOutsideTheDomainIsRejected(t *testing.T) {
	m := umpire.NewMachine[door, doorOutcome, doorFact]("test.door", "broken").
		Starts(door{Phase: closed}).
		Step0(push, func(door) []doorStep { return []doorStep{{Outcome: ok, State: door{Phase: "ajar"}}} })
	_, err := m.Table()
	require.EqualError(t, err, "broken: row closed-false-push lands in ajar-false, which is outside the state domain")
}

func TestTwoStepsBindingOneClassAreRejected(t *testing.T) {
	m := newDoor("twice").Step0(push, pushStep)
	_, err := m.Table()
	require.EqualError(t, err, `twice: two steps bind the action class "push"`)
}

func TestAMachineWithoutAStartIsRejected(t *testing.T) {
	m := umpire.NewMachine[door, doorOutcome, doorFact]("test.door", "nowhere").Step0(push, pushStep)
	_, err := m.Table()
	require.EqualError(t, err, "nowhere: the machine declares no start")
}

func TestAnExampleOfTheWrongTypeIsRejected(t *testing.T) {
	bad := umpire.NewAction1[hand]("wave", "person", "hand", umpire.Example("left", "Hello"))
	m := umpire.NewMachine[door, doorOutcome, doorFact]("test.door", "waving").
		Starts(door{Phase: closed}).Step1(bad, func(door, hand) []doorStep { return nil })
	_, err := m.Table()
	require.ErrorContains(t, err, "example left is a string, not a umpire_test.hand")
}

func TestCheckReportsAStuckStateAndAFactWithNoEvidence(t *testing.T) {
	// No lock: a closed door that cannot be opened by the left hand is still enabled by the right,
	// but an oiled open door can only be pushed shut, so remove push to leave open stuck.
	m := umpire.NewMachine[door, doorOutcome, doorFact]("test.door", "stuck").
		Starts(door{Phase: closed}).
		Ends(func(d door) bool { return d.Phase == locked }).
		Evidence("opened", "opened").
		Step1(turn, turnStep)
	err := umpire.Check(m)
	require.ErrorContains(t, err, "machine stuck: the machine reaches 'open-false', does not end there")
	require.ErrorContains(t, err, "row closed-false-turn-right-true records creaked-true, and no Evidence line names what confirms it")
}

// A two-phase abstraction of the door: shut or open.

type abstractPhase string

const (
	shut abstractPhase = "shut"
	ajar abstractPhase = "ajar"
)

func (abstractPhase) Values() []abstractPhase { return []abstractPhase{shut, ajar} }

type abstractDoor struct{ Phase abstractPhase }

type abstractFact string

const opens abstractFact = "opened"

func (abstractFact) Values() []abstractFact { return []abstractFact{opens} }

type abstractStep = umpire.Step[abstractDoor, doorOutcome, abstractFact]

var swing = umpire.NewAction0("turn", "person")

func abstractMachine(opensFact bool) *umpire.Machine[abstractDoor, doorOutcome, abstractFact] {
	return umpire.NewMachine[abstractDoor, doorOutcome, abstractFact]("test.door", "abstract").
		Starts(abstractDoor{shut}).
		Step0(swing, func(d abstractDoor) []abstractStep {
			if d.Phase != shut {
				return nil
			}
			var facts []abstractFact
			if opensFact {
				facts = []abstractFact{opens}
			}
			return []abstractStep{{Outcome: ok, State: abstractDoor{ajar}, Facts: facts}}
		}).
		Step0(push, func(d abstractDoor) []abstractStep {
			if d.Phase != ajar {
				return nil
			}
			return []abstractStep{{Outcome: ok, State: abstractDoor{shut}}}
		})
}

func abstractOf(d door) abstractDoor {
	if d.Phase == open {
		return abstractDoor{ajar}
	}
	return abstractDoor{shut}
}

func TestRefinementMatchesStepsAndStutters(t *testing.T) {
	abstract := abstractMachine(true)
	m := newDoor("concrete").Refines(abstract, abstractOf)
	ref, err := m.Refinement()
	require.NoError(t, err)
	byKey := map[string]*string{}
	for _, r := range ref.Rows {
		byKey[r.Key] = r.Product
	}
	require.Equal(t, "turn", *byKey["closed-false-turn-right-true"], "a turn is the abstract turn, by name")
	require.Equal(t, "push", *byKey["open-false-push"])
	require.Nil(t, byKey["closed-false-lock"], "locking stays shut: a stutter")
	require.Contains(t, tableOf(t, m).StateFields, "abstract")
}

func TestRefinementRejectsARowWithNoProductStep(t *testing.T) {
	abstract := umpire.NewMachine[abstractDoor, doorOutcome, abstractFact]("test.door", "abstract").
		Starts(abstractDoor{shut})
	m := newDoor("concrete").Refines(abstract, abstractOf)
	_, err := m.Refinement()
	require.ErrorContains(t, err, "the row 'closed-false-turn-right-true' steps from 'closed-false' to 'open-false', "+
		"which read as 'shut' and 'ajar' in abstract; abstract has no step")
}

func TestRefinementRejectsAProductFactTheRowDoesNotRecord(t *testing.T) {
	// The abstract turn records opened; a concrete turn that records nothing is not that step.
	abstract := abstractMachine(true)
	silent := umpire.NewMachine[door, doorOutcome, doorFact]("test.door", "silent").
		Starts(door{Phase: closed}).
		Step1(turn, func(d door, h hand) []doorStep {
			if d.Phase != closed {
				return nil
			}
			return []doorStep{{Outcome: ok, State: door{Phase: open, Oiled: d.Oiled}}}
		}).
		Refines(abstract, abstractOf)
	_, err := silent.Refinement()
	require.ErrorContains(t, err, "neither a step of abstract nor a stutter")
}

func TestRefinementRejectsAStartTheProductDoesNotHave(t *testing.T) {
	abstract := abstractMachine(true)
	m := umpire.NewMachine[door, doorOutcome, doorFact]("test.door", "opensFirst").
		Starts(door{Phase: open}).Step0(push, pushStep).
		Refines(abstract, abstractOf)
	_, err := m.Refinement()
	require.EqualError(t, err, "opensFirst refines abstract: opensFirst starts at 'open-false', which reads as "+
		"'ajar', and abstract does not start there")
}

// Queries on the door.

var (
	two  = umpire.Limits{Name: "two", Steps: 2, Actions: 2, Search: 64}
	tiny = umpire.Limits{Name: "tiny", Steps: 2, Actions: 2, Search: 1}
)

func TestFindReturnsTheShortestWitness(t *testing.T) {
	m := newDoor("door")
	opensLoudly := m.Property("opensLoudly").When(turn.With(right{Strong: true})).
		Holds(func(s doorStep) bool { return s.State.Phase == open })
	path := m.Scenario("turnThenPush").Starts(door{Phase: closed}).Actions(turn.With(right{Strong: true}), push.With())
	a, err := path.Find("q", opensLoudly, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.Found, a.Outcome)
	require.Equal(t, []string{"closed-false-turn-right-true", "open-false-push"}, a.Rows)
	require.Equal(t, "test.door.action.door.turn-right-true", a.Witness.Steps[0].Action.ID)
	require.Equal(t, []umpire.Atom{{ID: "test.door.fact.door.opened", Value: "opened"},
		{ID: "test.door.fact.door.creaked-true", Value: "creaked-true"}}, a.Witness.Steps[0].Facts)
}

func TestFindReportsNotFoundAndAWrongPathIsRejected(t *testing.T) {
	m := newDoor("door")
	never := m.Property("never").When(push.With()).Holds(func(s doorStep) bool { return s.State.Phase == open })
	path := m.Scenario("turnThenPush").Starts(door{Phase: closed}).Actions(turn.With(right{Strong: true}), push.With())
	a, err := path.Find("q", never, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.NotFound, a.Outcome)

	weak := m.Scenario("weakTurn").Starts(door{Phase: closed}).Actions(turn.With(left{}))
	a, err = weak.Find("q", never, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.NotFound, a.Outcome, "a pinned action with no row admits no trace")

	long := m.Scenario("long").Starts(door{Phase: closed}).
		Actions(turn.With(right{Strong: true}), push.With(), lock.With())
	_, err = long.Find("q", never, two).Answer()
	require.EqualError(t, err, "query q: long pins 3 actions and the limits two allow 2")
}

func TestVerifyFindsACounterexampleOrVerifies(t *testing.T) {
	m := newDoor("door")
	staysShut := m.Property("staysShut").HoldsAcross(func(before door, after doorStep) bool {
		return after.State.Phase != open
	})
	free := m.Scenario("anything").Starts(door{Phase: closed}).Free()
	a, err := free.Verify("q", staysShut, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.CounterexampleFound, a.Outcome)
	require.Equal(t, []string{"closed-false-turn-right-true"}, a.Rows)

	lockedIsFinal := m.Property("lockedIsFinal").HoldsAcross(func(before door, after doorStep) bool {
		return before.Phase != locked || after.State.Phase == locked
	})
	a, err = free.Verify("q", lockedIsFinal, two).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, a.Outcome)
}

func TestSearchStopsAtItsLimit(t *testing.T) {
	m := newDoor("door")
	p := m.Property("p").HoldsAcross(func(door, doorStep) bool { return true })
	a, err := m.Scenario("anything").Starts(door{Phase: closed}).Free().Verify("q", p, tiny).Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.LimitReached, a.Outcome)
}

func TestAFindCannotRealizeATransitionClaim(t *testing.T) {
	m := newDoor("door")
	p := m.Property("p").HoldsAcross(func(door, doorStep) bool { return true })
	_, err := m.Scenario("s").Starts(door{Phase: closed}).Actions(lock.With()).Find("q", p, two).Answer()
	require.EqualError(t, err, "query q: find names p, a transition claim; a find realizes a same-step claim")
}

func TestCoverageTargetsAreCutAtTheBudget(t *testing.T) {
	m := newDoor("door")
	targets, err := umpire.CoverageTargets(m, []umpire.CoverageGoal{umpire.CoverRows, umpire.CoverResults},
		umpire.Limits{Name: "b", Steps: 1, Search: 2})
	require.NoError(t, err)
	require.Equal(t, []umpire.CoverageTarget{
		{Kind: "row", Key: "closed-false-lock", State: "test.door.state.door.closed-false",
			Action: "test.door.action.door.lock", Results: []string{"test.door.outcome.door.ok"}},
		{Kind: "row", Key: "closed-false-turn-right-true", State: "test.door.state.door.closed-false",
			Action: "test.door.action.door.turn-right-true", Results: []string{"test.door.outcome.door.ok"}},
	}, targets)
}

// A composition of the door with a keyholder who may lose the key.

type keyPhase string

const (
	holding keyPhase = "holding"
	lost    keyPhase = "lost"
)

func (keyPhase) Values() []keyPhase { return []keyPhase{holding, lost} }

type keyState struct{ Phase keyPhase }

type keyFact interface{ isKeyFact() }

var _ = umpire.Sum[keyFact]()

type keyStep = umpire.Step[keyState, doorOutcome, keyFact]

var (
	useKey  = umpire.NewAction0("useKey", "person")
	loseKey = umpire.NewAction0("loseKey", "person")
)

var keyholder = umpire.NewMachine[keyState, doorOutcome, keyFact]("test.door", "keyholder").
	Starts(keyState{holding}).
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

type house struct {
	Door door     `umpire:"door"`
	Key  keyState `umpire:"key"`
}

func TestCompositionSynchronizesAndKeysByMember(t *testing.T) {
	c := umpire.Compose[house]("test.door", "house").
		Member("door", newDoor("door")).
		Member("key", keyholder).
		Sync("lock", "door.lock", "key.useKey")
	tb := tableOf(t, c)
	require.Equal(t, []string{"door_push", "door_turn-left", "door_turn-right-false", "door_turn-right-true",
		"key_loseKey", "lock"}, tb.Actions)
	require.Contains(t, tb.States, "closed-false_holding")
	require.Contains(t, tb.States, "locked-false_holding")
	for _, r := range tb.Rows {
		if r.Action == "lock" {
			require.NotContains(t, r.Source, "_lost", "a lost key cannot lock the door")
		}
	}
	require.Equal(t, "test.door.target.compose-house", tb.IDs().Target)
	require.Equal(t, []string{"door_phase", "door_oiled", "key"}, tb.StateFields)

	bad := umpire.Compose[house]("test.door", "bad").Member("door", newDoor("door")).Sync("x", "door", "key.useKey")
	_, err := bad.Table()
	require.EqualError(t, err, `compose-bad: sync reference "door" is not <member>.<action>`)
}

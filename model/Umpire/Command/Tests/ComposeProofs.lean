import Umpire.Command.Tests.Compose

/-!
# What a composed table agrees with

The agreement check over tables by position, on tables small enough to read: two members, a lamp
that lights and a switch that flips, composed so that the lamp lights only when the switch flips.
The literal the walk would emit passes the check; a literal with a row the members do not
authorize, one missing a row they do, one keeping a state no start reaches, one whose row steps to
a state outside its catalog, and one starting where a member does not fail it, each on the clause
the failure is about. The command's refusal is pinned through `elabComposedAgreement` over one of
those literals, since the walk itself emits nothing the check refuses, and the axioms of the
generic theorem and of a fixture composition's own agreement theorem are pinned.
-/

namespace Umpire.Command.Tests.ComposeProofs

open Umpire.Command.Compose

/-! ### Two members by position

The lamp: states `dark` (0) and `lit` (1), actions `light` (0) and `flip` (1), outcome `accepted`
(0), fact `shone` (0). It lights from dark, recording that it shone, and a flip leaves it where
it is. The switch: states `off` (0) and `on` (1), actions `flip` (0) and `light` (1). It flips
either way, and lights only while on. -/

private def lamp : IndexedMember := {
  starts := [0]
  rows := [
    [(0, [{ outcome := 0, state := 1, facts := [0] }]), (1, [{ outcome := 0, state := 0, facts := [] }])],
    [(1, [{ outcome := 0, state := 1, facts := [] }])]] }

private def switch : IndexedMember := {
  starts := [0]
  rows := [
    [(0, [{ outcome := 0, state := 1, facts := [] }])],
    [(0, [{ outcome := 0, state := 0, facts := [] }]), (1, [{ outcome := 0, state := 1, facts := [] }])]] }

/-- `flip` synchronizes both flips; `light` synchronizes the lamp's light with the switch's. -/
private def flip : List (Nat × Nat) := [(0, 1), (1, 0)]
private def light : List (Nat × Nat) := [(0, 0), (1, 1)]

/-- From `[dark, off]`: a flip reaches `[dark, on]`, a light there reaches `[lit, on]`, and flips
move the switch under the lamp; `light` has no row while the switch is off. -/
private def literal : IndexedLiteral := {
  starts := [[0, 0]]
  states := [[0, 0], [0, 1], [1, 0], [1, 1]]
  actions := [flip, light]
  rows := [
    { source := [0, 0], participants := flip, results := [{ outcome := (0, 0), state := [0, 1], facts := [] }] },
    { source := [0, 1], participants := flip, results := [{ outcome := (0, 0), state := [0, 0], facts := [] }] },
    { source := [0, 1], participants := light, results := [{ outcome := (0, 0), state := [1, 1], facts := [(0, 0)] }] },
    { source := [1, 1], participants := flip, results := [{ outcome := (0, 0), state := [1, 0], facts := [] }] },
    { source := [1, 0], participants := flip, results := [{ outcome := (0, 0), state := [1, 1], facts := [] }] }] }

#guard composedTableAgrees [lamp, switch] literal

/- The composition function at the start: a flip moves the switch, and a light has no row. -/
#guard stepsFrom [lamp, switch] [0, 0] flip ==
  some [{ outcome := (0, 0), state := [0, 1], facts := [] }]
#guard stepsFrom [lamp, switch] [0, 0] light == none

/- The order the rows list results in is not the check's: a row holds the same results either
way. -/
#guard composedTableAgrees [lamp, switch]
  { literal with rows := literal.rows.map fun row => { row with results := row.results.reverse } }

/-! ### What the check refuses -/

/- A row the members do not authorize: a light while the switch is off. -/
#guard !composedTableAgrees [lamp, switch] { literal with rows := literal.rows ++
  [{ source := [0, 0], participants := light,
     results := [{ outcome := (0, 0), state := [1, 0], facts := [(0, 0)] }] }] }

/- A row with the wrong result: the light's fact left out. -/
#guard !composedTableAgrees [lamp, switch] { literal with rows := literal.rows.map fun row =>
  if row.participants == light then { row with results := [{ outcome := (0, 0), state := [1, 1], facts := [] }] }
  else row }

/- A row the members authorize, missing: the flip from `[lit, on]`. -/
#guard !composedTableAgrees [lamp, switch]
  { literal with rows := literal.rows.filter (·.source != [1, 1]) }

/- A state no start reaches, kept in the catalog: one beyond both members' catalogs, which no row
leaves or enters, so every row still agrees. -/
#guard !composedTableAgrees [lamp, switch] { literal with states := literal.states ++ [[2, 2]] }

/- A row stepping outside the catalog: `[lit, off]` dropped from the states. -/
#guard !composedTableAgrees [lamp, switch]
  { literal with states := literal.states.filter (· != [1, 0]) }

/- A start no member starts in: the switch on. -/
#guard !composedTableAgrees [lamp, switch] { literal with starts := [[0, 1]] }

/- A start of the wrong width. -/
#guard !composedTableAgrees [lamp, switch] { literal with starts := [[0]] }

/-! ### What the command does with it

The refusal is at the theorem the command declares, so it is pinned by declaring one over the
literal missing a row; a composition the walk emits is never that literal. The refused theorem
stays declared carrying `sorryAx`, as a machine's unproven table does, and the command's error is
what keeps a refused composition from being a Model. -/

/--
error: the literal table of 'lamped', 4 states over 2 actions, is not the composition of its members over the states its starts reach, so the kernel refused its agreement theorem; the walk emitted a row the members do not authorize, missed one they do, or kept a state no start reaches
-/
#guard_msgs (error, substring := true) in
run_cmd do
  let members ← `(term| [Umpire.Command.Tests.ComposeProofs.lamp,
    Umpire.Command.Tests.ComposeProofs.switch])
  let missingRow ← `(term| { Umpire.Command.Tests.ComposeProofs.literal with
    rows := Umpire.Command.Tests.ComposeProofs.literal.rows.filter (·.source != [1, 1]) })
  Umpire.Command.elabComposedAgreement .missing "lamped" (Lean.mkIdent `lampedRefused) members
    missingRow 4 2

/-- info: 'Umpire.Command.Tests.ComposeProofs.lampedRefused' depends on axioms: [propext, sorryAx, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms lampedRefused

run_cmd do
  let members ← `(term| [Umpire.Command.Tests.ComposeProofs.lamp,
    Umpire.Command.Tests.ComposeProofs.switch])
  let literal ← `(term| Umpire.Command.Tests.ComposeProofs.literal)
  Umpire.Command.elabComposedAgreement .missing "lamped" (Lean.mkIdent `lampedAgrees) members
    literal 4 2

/-- info: 'Umpire.Command.Tests.ComposeProofs.lampedAgrees' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms lampedAgrees

/-! ### The axioms

The generic theorem, and the agreement theorem `compose` declared for the fixture composition. -/

/-- info: 'Umpire.Command.Compose.ComposedAgreement.ofChecked' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Command.Compose.ComposedAgreement.ofChecked

/-- info: 'Umpire.Command.Tests.Compose.Forward.pipeline.agrees' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Command.Tests.Compose.Forward.pipeline.agrees

/- The fixture composition's literal, read through the view the command generated, is what the
check saw: 12 states, 7 actions, and its rows by position. -/
#guard (IndexedLiteral.ofModel Compose.Forward.pipeline Compose.Forward.pipeline.view).states.length
  == 12
#guard (IndexedLiteral.ofModel Compose.Forward.pipeline Compose.Forward.pipeline.view).actions ==
  [[(1, 1)], [(0, 1), (1, 0)], [(0, 0)], [(0, 2)], [(0, 3)], [(0, 4), (1, 2)], [(0, 5), (1, 2)]]

end Umpire.Command.Tests.ComposeProofs

import Umpire.Command.Tests.Compose
import Umpire.Shared.Test

/-!
# What a composed table agrees with

The agreement check over tables by position, on tables small enough to read: two members, a lamp
that lights and a switch that flips, composed so that the lamp lights only when the switch flips.
The literal the walk would emit, grouped by state as the check reads it, passes the check; a
literal with a row the members do not authorize, one missing a row they do, one missing an enabled
action with its rows and the states only it reaches, one whose catalog holds an action that is no
candidate, one keeping a state no start reaches, one whose row steps to a state outside its
catalog, and one starting where a member does not fail it, each on the clause the failure is
about. The grouping is pinned to flatten back to the literal it was read from, which is what the
command decides of it. The command's refusal is pinned through `elabComposedAgreement` over one of
those literals, since the walk itself emits nothing the check refuses, and the axioms of the
generic theorem and of a fixture composition's own agreement theorem are pinned, with the literals
the command declared beside it.
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

/- The candidates, read off every action of a composed domain: the two synchronized actions, and
no member's own action, since each is named by one of them. -/
#guard candidatesOf [[(0, 0)], [(0, 1)], [(1, 0)], [(1, 1)], flip, light] == [flip, light]

/-- From `[dark, off]`: a flip reaches `[dark, on]`, a light there reaches `[lit, on]`, and flips
move the switch under the lamp; `light` has no row while the switch is off. The rows are in state
catalog order, as the walk emits them. -/
private def literal : IndexedLiteral := {
  starts := [[0, 0]]
  states := [[0, 0], [0, 1], [1, 0], [1, 1]]
  actions := [flip, light]
  rows := [
    { source := [0, 0], participants := flip, results := [{ outcome := (0, 0), state := [0, 1], facts := [] }] },
    { source := [0, 1], participants := flip, results := [{ outcome := (0, 0), state := [0, 0], facts := [] }] },
    { source := [0, 1], participants := light, results := [{ outcome := (0, 0), state := [1, 1], facts := [(0, 0)] }] },
    { source := [1, 0], participants := flip, results := [{ outcome := (0, 0), state := [1, 1], facts := [] }] },
    { source := [1, 1], participants := flip, results := [{ outcome := (0, 0), state := [1, 0], facts := [] }] }] }

/-- The literal grouped by state: the start at position 0, and each state's rows under its
position, every result's state a position. -/
private def grouped : GroupedLiteral := {
  starts := [0]
  states := literal.states
  actions := literal.actions
  rows := [
    [{ participants := flip, results := [{ outcome := (0, 0), state := 1, facts := [] }] }],
    [{ participants := flip, results := [{ outcome := (0, 0), state := 0, facts := [] }] },
     { participants := light, results := [{ outcome := (0, 0), state := 3, facts := [(0, 0)] }] }],
    [{ participants := flip, results := [{ outcome := (0, 0), state := 3, facts := [] }] }],
    [{ participants := flip, results := [{ outcome := (0, 0), state := 2, facts := [] }] }]] }

#guard GroupedLiteral.ofFlat literal == grouped
#guard grouped.flatten == literal

/- A literal whose rows are not in state order does not flatten back to itself, so the command's
decision that the grouping flattens to the literal refuses it. -/
#guard (GroupedLiteral.ofFlat { literal with rows := literal.rows.reverse }).flatten !=
  { literal with rows := literal.rows.reverse }

#guard composedTableAgrees [lamp, switch] [flip, light] grouped

/- The composition function at the start: a flip moves the switch, and a light has no row. -/
#guard stepsFrom [lamp, switch] [0, 0] flip ==
  some [{ outcome := (0, 0), state := [0, 1], facts := [] }]
#guard stepsFrom [lamp, switch] [0, 0] light == none

/- The order the rows list results in is not the check's: a row holds the same results either
way. -/
#guard composedTableAgrees [lamp, switch] [flip, light]
  (.ofFlat { literal with rows := literal.rows.map fun row => { row with results := row.results.reverse } })

/-! ### What the check refuses -/

/- A row the members do not authorize: a light while the switch is off. -/
#guard !composedTableAgrees [lamp, switch] [flip, light] (.ofFlat { literal with rows := literal.rows ++
  [{ source := [0, 0], participants := light,
     results := [{ outcome := (0, 0), state := [1, 0], facts := [(0, 0)] }] }] })

/- A row with the wrong result: the light's fact left out. -/
#guard !composedTableAgrees [lamp, switch] [flip, light] (.ofFlat { literal with rows := literal.rows.map fun row =>
  if row.participants == light then { row with results := [{ outcome := (0, 0), state := [1, 1], facts := [] }] }
  else row })

/- A row the members authorize, missing: the flip from `[lit, on]`. -/
#guard !composedTableAgrees [lamp, switch] [flip, light]
  (.ofFlat { literal with rows := literal.rows.filter (·.source != [1, 1]) })

/- An enabled action dropped whole: `light` out of the catalog with its row and the two lit states
only it reaches. What remains agrees with itself, and the candidates are what say it is short. -/
#guard !composedTableAgrees [lamp, switch] [flip, light] (.ofFlat
  { starts := [[0, 0]], states := [[0, 0], [0, 1]], actions := [flip],
    rows := literal.rows.filter fun row => row.participants == flip && row.source[0]? == some 0 })
#guard composedTableAgrees [lamp, switch] [flip] (.ofFlat
  { starts := [[0, 0]], states := [[0, 0], [0, 1]], actions := [flip],
    rows := literal.rows.filter fun row => row.participants == flip && row.source[0]? == some 0 })

/- A catalog action that is no candidate. -/
#guard !composedTableAgrees [lamp, switch] [flip] grouped

/- A state no start reaches, kept in the catalog: one beyond both members' catalogs, which no row
leaves or enters, so every row still agrees. -/
#guard !composedTableAgrees [lamp, switch] [flip, light]
  (.ofFlat { literal with states := literal.states ++ [[2, 2]] })

/- A row stepping outside the catalog: `[lit, off]` dropped from the states. -/
#guard !composedTableAgrees [lamp, switch] [flip, light]
  (.ofFlat { literal with states := literal.states.filter (· != [1, 0]) })

/- A start no member starts in: the switch on. -/
#guard !composedTableAgrees [lamp, switch] [flip, light] (.ofFlat { literal with starts := [[0, 1]] })

/- A start of the wrong width. -/
#guard !composedTableAgrees [lamp, switch] [flip, light] (.ofFlat { literal with starts := [[0]] })

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
  let candidates ← `(term| [Umpire.Command.Tests.ComposeProofs.flip,
    Umpire.Command.Tests.ComposeProofs.light])
  Umpire.Command.elabComposedAgreement .missing "lamped" (Lean.mkIdent `lampedRefused) members
    candidates missingRow 4 2

-- `lampedRefused` stays declared carrying `sorryAx` (the comment above), and `assert_axioms`
-- rejects `sorryAx` unconditionally, so the checker is expected to reject this declaration --
-- proving the refusal actually leaves it unproven rather than silently accepting it.
/-- still contains `sorry` -/
#guard_msgs (error, substring := true) in
assert_axioms [lampedRefused] allowing [propext, Classical.choice, Quot.sound]

run_cmd do
  let members ← `(term| [Umpire.Command.Tests.ComposeProofs.lamp,
    Umpire.Command.Tests.ComposeProofs.switch])
  let candidates ← `(term| [Umpire.Command.Tests.ComposeProofs.flip,
    Umpire.Command.Tests.ComposeProofs.light])
  let literal ← `(term| Umpire.Command.Tests.ComposeProofs.literal)
  Umpire.Command.elabComposedAgreement .missing "lamped" (Lean.mkIdent `lampedAgrees) members
    candidates literal 4 2

/-! ### The axioms

The generic theorem, and the agreement theorem `compose` declared for the fixture composition. -/

assert_axioms [lampedAgrees, Umpire.Command.Compose.ComposedAgreement.ofChecked,
  Umpire.Command.Tests.Compose.Forward.pipeline.agrees]
  allowing [propext, Classical.choice, Quot.sound]

/- The literals the command declared beside the theorem: the readings the kernel decided equal
the tables' and the composed literal's, and the grouping the check ran on, which flattens to the
literal. -/
#guard Compose.Forward.pipeline.agrees.members ==
  [IndexedMember.ofModel Compose.Job.jobMachine, IndexedMember.ofModel Compose.Agent.agentMachine]
#guard Compose.Forward.pipeline.agrees.literal ==
  IndexedLiteral.ofModel Compose.Forward.pipeline Compose.Forward.pipeline.view
#guard Compose.Forward.pipeline.agrees.grouped.flatten == Compose.Forward.pipeline.agrees.literal

/- A member's table by position groups its rows under their source: the agent, running (0) or
halted (1), halts (0) and serves (2) while running and resumes (1) while halted. -/
#guard IndexedMember.ofModel Compose.Agent.agentMachine == {
  starts := [0]
  rows := [
    [(0, [{ outcome := 0, state := 1, facts := [] }]), (2, [{ outcome := 0, state := 0, facts := [] }])],
    [(1, [{ outcome := 0, state := 0, facts := [] }])]] }

/- The fixture composition's literal, read through the view the command generated, is what the
check saw: 12 states, 7 actions, and its rows by position. Its candidates, read off the Action
domain in the domain's own order, are its seven catalog actions: the job's two pokes and its
expire, the agent's resume, the halt, and the two replies. -/
#guard (IndexedLiteral.ofModel Compose.Forward.pipeline Compose.Forward.pipeline.view).states.length
  == 12
#guard (IndexedLiteral.ofModel Compose.Forward.pipeline Compose.Forward.pipeline.view).actions ==
  [[(1, 1)], [(0, 1), (1, 0)], [(0, 0)], [(0, 2)], [(0, 3)], [(0, 4), (1, 2)], [(0, 5), (1, 2)]]
#guard candidatesOf ((Umpire.Command.members (α := Compose.Forward.pipeline.Action)).map
    Compose.Forward.pipeline.view.action) ==
  [[(0, 3)], [(0, 2)], [(0, 0)], [(1, 1)], [(0, 1), (1, 0)], [(0, 5), (1, 2)], [(0, 4), (1, 2)]]

/- The gates composition's catalog left `early` out as never enabled; the candidates keep it, and
the theorem holds because no reachable state enables it. -/
#guard (candidatesOf ((Umpire.Command.members (α := Compose.gates.Action)).map
    Compose.gates.view.action)).length == 3 &&
  (IndexedLiteral.ofModel Compose.gates Compose.gates.view).actions.length == 2

end Umpire.Command.Tests.ComposeProofs

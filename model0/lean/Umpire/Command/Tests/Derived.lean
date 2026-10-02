import Umpire.Command

/-!
# Derived machines: `from:`, `restrict:` and `extend:`

A lamp is pressed on at one of two levels, released off, and burns out on a timer that records
nothing. Machines derived from it keep some of its actions, add results to one, or both, and each
is read back through what the command generated for it: its catalogs, its rows, the evidence and
timers it kept, and the Definition IDs it owns. A refining machine shows that a derived machine
inherits neither `refines:` nor the abstract state field. The rest are the located errors.
-/

namespace Umpire.Command.Tests.Derived

open Umpire Umpire.Command
open Lean Elab Command

model_conventions root "umpire" under Umpire.Command.Tests.Derived

/-! ### The source -/

entity lamp
  key: lampId

enum Level
  | low
  | high

enum Mode
  | off
  | on
  | broken

structure LampState where
  mode : Mode
  deriving BEq, DecidableEq, Repr, Finite

enum LampOutcome
  | accepted

enum LampFact
  | lit
  | dimmed
  | burnt

action press
  party: user
  on: lamp
  input:
    level: Level

action release
  party: user
  on: lamp

/-- A lamp that is off comes on at either level. -/
def pressStep (state : LampState) (_ : Level) : List (Step LampState LampOutcome LampFact) :=
  if state.mode != .off then [] else
  [{ outcome := .accepted, state := { mode := .on }, facts := [.lit] }]

/-- A lamp that is on goes off. -/
def releaseStep (state : LampState) : List (Step LampState LampOutcome LampFact) :=
  if state.mode != .on then [] else
  [{ outcome := .accepted, state := { mode := .off }, facts := [.dimmed] }]

/-- A lamp that is on burns out, and nothing records it. -/
def burnStep (state : LampState) : List (Step LampState LampOutcome LampFact) :=
  if state.mode != .on then [] else
  [{ outcome := .accepted, state := { mode := .broken }, facts := [] }]

machine lampMachine
  for: lamp
  state: LampState
  starts: [off]
  ends: [off, on, broken]
  timers: [burn]
  unobservable: [burn]
  evidence:
    lit: lit
    dimmed: dimmed
    burnt: burnt
  steps:
    press: pressStep
    release: releaseStep
    burn: burnStep

/-- What the registry recorded for a machine: its steps, timers, unobservable timers, evidence and
the machine it refines. -/
private def recorded (declName : Name) : CommandElabM Unit := do
  let some entry := Registry.machine? (← getEnv) declName
    | throwError "no machine {declName}"
  logInfo m!"steps {entry.steps.map (·.1)}, timers {entry.timers}, \
    unobservable {entry.unobservable}, evidence {entry.evidence.map (·.1)}, \
    refines {entry.refines}"

/-! ### `restrict:` -/

/- A lamp that is only pressed: release and the timer are gone from the catalog, so is the evidence
for the fact only release returned, and the burnt line, which no row returns. -/
machine pressOnly
  from: lampMachine
  restrict: [press]

#guard pressOnly.actionKeys == #["press-high", "press-low"]
#guard pressOnly.stateKeys == lampMachine.stateKeys

/-- info: steps [press], timers [], unobservable [], evidence [lit], refines none -/
#guard_msgs in
#eval recorded ``pressOnly

/- Its Definition IDs hang off its own name, not the source's. -/
#guard pressOnly.targetId.value == "umpire.target.pressOnly"
#guard pressOnly.stateIds.map (·.value) ==
  ["umpire.state.pressOnly.off", "umpire.state.pressOnly.on", "umpire.state.pressOnly.broken"]

/-! ### `extend:` -/

/-- A high press on a lamp that is off may also burn it out. -/
def overPress (state : LampState) (level : Level) : List (Step LampState LampOutcome LampFact) :=
  if state.mode != .off then [] else
  match level with
  | .high => [{ outcome := .accepted, state := { mode := .broken }, facts := [.burnt] }]
  | .low => []

/- The extension's result sorts before the source's, `broken` before `on`, so the row holds it
first although the source's comes first in the concatenation. -/
machine fragileLamp
  from: lampMachine
  extend:
    press: overPress

#guard (fragileLamp.step { mode := .off } (.press .high)).map (·.state.mode) == [.broken, .on]
#guard (fragileLamp.step { mode := .off } (.press .low)).map (·.state.mode) == [.on]
#guard fragileLamp.actionKeys == lampMachine.actionKeys

/- Everything else is the source's: the timer, still silent, and every evidence line, since the
extension now returns `burnt`. -/
/--
info: steps [press, release, burn], timers [burn], unobservable [burn], evidence [lit, dimmed, burnt], refines none
-/
#guard_msgs in
#eval recorded ``fragileLamp

/-! ### Both, restriction first -/

machine fragilePress
  from: lampMachine
  restrict: [press]
  extend:
    press: overPress

#guard fragilePress.actionKeys == #["press-high", "press-low"]
#guard (fragilePress.step { mode := .off } (.press .high)).map (·.state.mode) == [.broken, .on]

/-- info: steps [press], timers [], unobservable [], evidence [lit, burnt], refines none -/
#guard_msgs in
#eval recorded ``fragilePress

/-! ### A refining source

The refinement is the source's claim about its own table; a derived machine has another table, so
it carries neither the `refines:` nor the state field the refinement adds. -/

enum Glow
  | dark
  | bright

structure GlowState where
  phase : Glow
  deriving BEq, DecidableEq, Repr, Finite

def glowPressStep (state : GlowState) (_ : Level) : List (Step GlowState LampOutcome LampFact) :=
  if state.phase != .dark then [] else
  [{ outcome := .accepted, state := { phase := .bright }, facts := [.lit] }]

machine glow
  for: lamp
  state: GlowState
  starts: [dark]
  ends: [dark, bright]
  steps:
    press: glowPressStep

def glowOf (state : LampState) : GlowState :=
  { phase := if state.mode == .on then .bright else .dark }

machine glowingLamp
  for: lamp
  state: LampState
  refines: glow
  map: glowOf
  starts: [off]
  ends: [off, on, broken]
  steps:
    press: pressStep

machine plainLamp
  from: glowingLamp

#guard glowingLamp.stateFieldIds.map (·.1) == ["mode", "glow"]
#guard plainLamp.stateFieldIds.map (·.1) == ["mode"]

/-- info: steps [press], timers [], unobservable [], evidence [], refines none -/
#guard_msgs in
#eval recorded ``plainLamp

/-! ### Located errors -/

/--
error: 'burn' is a timer; `restrict:` names the actions a derived machine keeps, and a timer is the source's system behaviour rather than an action a composition synchronizes
-/
#guard_msgs in
machine keepsTimer
  from: lampMachine
  restrict: [burn]

/--
error: the machine this one derives from does not step on 'poke', so `restrict:` cannot name it; it steps on 'press', 'release', 'burn'
-/
#guard_msgs in
machine keepsAbsent
  from: lampMachine
  restrict: [poke]

/-- error: `restrict:` names 'press' twice -/
#guard_msgs in
machine keepsTwice
  from: lampMachine
  restrict: [press, press]

/- `restrict:` applies first, so an extension of an action it dropped names nothing. -/
/--
error: the machine this one derives from does not step on 'press', so `extend:` cannot name it; it steps on 'release'
-/
#guard_msgs in
machine extendsDropped
  from: lampMachine
  restrict: [release]
  extend:
    press: overPress

/-- error: `extend:` names 'press' twice -/
#guard_msgs in
machine extendsTwice
  from: lampMachine
  extend:
    press: overPress
    press: overPress

/-- A press on a broken lamp mends it, where the source has no press row. -/
def mendPress (state : LampState) (_ : Level) : List (Step LampState LampOutcome LampFact) :=
  if state.mode != .broken then [] else
  [{ outcome := .accepted, state := { mode := .off }, facts := [] }]

/--
error: the extension of 'press' returns a result at state 'broken' for 'press-high', where the source has no row; `extend:` adds results to the source's rows and never enables an action
-/
#guard_msgs in
machine mended
  from: lampMachine
  extend:
    press: mendPress

/--
error: the extension of 'press' returns a result the row at state 'off' for 'press-high' already holds, or returns it twice
-/
#guard_msgs in
machine pressedTwice
  from: lampMachine
  extend:
    press: pressStep

/--
error: `restrict:` derives a machine from another, so it needs a `from:` line naming the machine it derives from
-/
#guard_msgs in
machine restrictsNothing
  for: lamp
  state: LampState
  starts: [off]
  ends: [off, on, broken]
  restrict: [press]
  steps:
    press: pressStep

/--
error: a machine with `from:` takes its entity, state, starts, ends, steps, evidence and timers from the machine it derives from; it adds only `restrict:` and `extend:`
-/
#guard_msgs in
machine redeclared
  from: lampMachine
  starts: [off]

/--
error: 'pressStep' is not a machine declared by a `machine` command; `from:` names the machine this one derives from
-/
#guard_msgs in
machine fromFunction
  from: pressStep

end Umpire.Command.Tests.Derived

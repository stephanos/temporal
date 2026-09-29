import Temporal.Case.Syntax

/-!
# The `schema:` and `evidence:` checks reach a production Model's own import

A production Model imports only `Temporal.Case.Syntax`, the way
`Temporal.Feature.System.Info.Model` does. `Umpire.Command.checkSchema` and
`Umpire.Command.checkCatalog` accept every name when no platform module installed a check, so an
unresolvable `schema:` message or an uncatalogued `evidence:` observation is accepted silently
unless importing `Temporal.Case.Syntax` alone is enough to turn both checks on. The two pinned
rejections below are that proof: each fires from this file's own import, with no further import of
`Temporal.Case.Schema` or `Temporal.Case.Catalog`.
-/

namespace Temporal.Case.Tests.ProductionImports

open Umpire
open Umpire.Command

entity widget

enum WidgetPhase
  | idle
  | done

structure WidgetState where
  phase : WidgetPhase
  deriving BEq, DecidableEq, Repr, Finite

enum WidgetOutcome
  | accepted

enum WidgetFact
  | somethingHappened

action tick
  party: caller
  on: widget

def tickStep (state : WidgetState) :
    List (Step WidgetState WidgetOutcome WidgetFact) :=
  if state.phase != .idle then [] else
  [{ outcome := .accepted, state := { phase := .done }, facts := [.somethingHappened] }]

/- A `schema:` name resolves against the generated API only if this file's own import turns the
check on. -/
/--
error: 'temporal.api.workflowservice.v1.NoSuchMessage' does not resolve to a protobuf message the generated API carries
-/
#guard_msgs in
action badSchema
  party: caller
  on: widget
  schema: temporal.api.workflowservice.v1.NoSuchMessage

/- An `evidence:` observation is checked against the realization's catalog only if this file's own
import turns the check on. -/
/--
error: 'notCatalogued' is neither a recorded event kind the realization carries, a read observation its catalog binds, nor an observation this Model declares; evidence names recorded data, so it is a generated history event kind, a Testpilot Run Event kind, a catalog read such as `pendingAttempts`, or a derived `observation`
-/
#guard_msgs in
machine widgetMachine
  for: widget
  state: WidgetState
  starts: [idle]
  ends: [done]
  evidence:
    somethingHappened: notCatalogued
  steps:
    tick: tickStep

end Temporal.Case.Tests.ProductionImports

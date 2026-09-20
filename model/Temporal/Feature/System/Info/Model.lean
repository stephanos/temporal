import Temporal.Case.Syntax

/-!
# The system-info Model

One unary workflow-service call, and the one thing the product promises about it: the server
answers it. It is the Model of a call that starts no workflow and leaves no history: what confirms
the step is the completion the runtime records for the instruction that made it. The realization
reads the server version out of the response into an observation beside the Contract.
-/

namespace Temporal.Feature.System.Info

open Umpire
open Umpire.Command

/-! ### Entities and domains -/

/-- The server a Run asks: one, named by itself. -/
entity server

enum Phase
  | unqueried
  | queried

structure InfoState where
  phase : Phase
  deriving BEq, DecidableEq, Repr, Finite

enum InfoOutcome
  | accepted

enum InfoFact
  | systemInfoReturned

/-! ### The action -/

action getSystemInfo
  party: caller
  on: server
  schema: temporal.api.workflowservice.v1.GetSystemInfoRequest

/-! ### The machine -/

/-- An unqueried server answers once, and the call's completion records it. -/
def queryStep (state : InfoState) :
    List (Step InfoState InfoOutcome InfoFact) :=
  if state.phase != .unqueried then [] else
  [{ outcome := .accepted, state := { phase := .queried }, facts := [.systemInfoReturned] }]

machine systemInfo
  for: server
  state: InfoState
  starts: [unqueried]
  ends: [queried]
  evidence:
    systemInfoReturned: instructionCompleted
  steps:
    getSystemInfo: queryStep

/-! ### What the machine promises -/

/- The call settles the server as queried, and its completion records it. -/
property infoReturned
  machine: systemInfo
  when: getSystemInfo
  holds: fun step =>
    step.state.phase == .queried && step.facts.contains .systemInfoReturned

/-! ### The path, the Query and the set -/

scenario once
  model: systemInfo
  starts: unqueried
  actions: [getSystemInfo]

limits one
  steps: 1
  actions: 1
  search: 8

query answered
  find: infoReturned
  in: once
  limits: one

set systemInfoTests
  purpose: functional
  bind:
    caller: driven
  queries: [answered]

case systemInfoCases
  realizes systemInfoTests
  as Temporal.Case.Realization.unaryRpc

end Temporal.Feature.System.Info

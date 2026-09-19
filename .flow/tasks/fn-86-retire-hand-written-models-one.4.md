---
satisfies: [R6]
---
# fn-86-retire-hand-written-models-one.4 Re-anchor the Implementation Link on the Nexus product machine

## Description
`Temporal.System.Nexus.ImplementationLink` is the kept SEM-08 link, but it imports `Temporal.Feature.Nexus.Lifecycle` (about fifteen element definitions pinned by `native_decide` plus four authoritative lemmas) and `Temporal.Feature.Nexus.Race.Terminal` (the terminal-closure projection), and its evidence tests import `Operations`; all three are on the deletion list. Before task .5 deletes anything, re-anchor the link on fn-85's Nexus product machine with the forward simulation still proved and the terminal closure still checked, and re-express the link's evidence pins on the Caller Model's Queries.

**Size:** M
**Files:** `model/Temporal/System/Nexus/ImplementationLink.lean` (imports and element definitions from the Caller Model's `nexusProduct`; the terminal-closure input from the product machine's `ends:`), `model/Temporal/System/Nexus/ImplementationLinkTests.lean`, `model/TemporalModelTests/Nexus/ImplementationLink.lean` (evidence pins over the Caller Model's Queries instead of `Operations.*.property` and `intendedTrace`), `model/Temporal/Feature/Nexus/Caller/Model.lean` (only if an element definition needs a stable name), `model/ModelLint/ImportGraph.lean` (the `implementationLinkConsumers` entry stays single; verify)
**Touches:** [model/Temporal/System/Nexus/**, model/TemporalModelTests/**, model/Temporal/Feature/Nexus/Caller/Model.lean, model/ModelLint/ImportGraph.lean]

### Approach
- Map each pinned Lifecycle element (states, actions, outcomes, observations, the four `target_*_authoritative` lemmas) to the product machine's declarations; where the product machine has no counterpart (an observation the old model declared), record a Known Gap in the link's required coverage rather than a shim, and list it in the receipt.
- `Race.Terminal.targetResult` fed the terminal-closure projection; the product machine's `ends:` values are the same information; rebuild the projection from them.
- The forward simulation must still be proved (`checkImplementationLink` returns `.complete`); its witness may be synthesized the way fn-85's refinement does (`decide`/`native_decide`) if the old hand proof no longer applies; record elaboration time.
- MOD-10's single exception stays `Temporal.System.Nexus.ImplementationLink`; the new lint rule (task .8) leaves it outside.
- Adjusted 2026-09-19 after fn-85 .14, .16 and .6 landed (fn-85 .10 still open). Until .10
  promotes it, `nexusProduct` lives in `model/Temporal/Feature/Nexus/Tests/Machines.lean:105-123`
  (a test module; anchor on the Caller Model once .10 lands, never on the test module). The product
  machine has a `timeout` timer out of `scheduled` and `started` (fn-85 .6) and a required `ends:`
  list, and Search takes steps out of end states, so "terminal" for the closure projection is
  `ends:` membership, not the absence of successors. fn-85 .6 added
  `Umpire.ImplementationLink.Refinement` (`RefinementMorphism`, `RefinedStep`,
  `StutteringSimulation`, `TableRefinement.ofChecked (by decide +kernel)`) as the witness shape a
  `machine` synthesizes; the link may reuse those obligations for its witness, but it stays the
  SEM-08 Implementation Link, not a `refines:` (the spec and DESIGN.md section 6 both say so). A
  product Property is read on the protocol machine through the `nexusProduct` state field (.6),
  which is the evidence pins' route to the Caller Model's Queries.

### Investigation targets
**Required:**
- `model/Temporal/System/Nexus/ImplementationLink.lean:1-2,35-350,484,528` — imports, pinned elements, the Terminal use, the closure projection
- `model/TemporalModelTests/Nexus/ImplementationLink.lean:1,302-354` — the evidence pins over Operations
- `model/Umpire/ImplementationLink/Language.lean:445,520-620,1160` — the simulation, `traceForward` and `checkImplementationLink`; `model/Umpire/ImplementationLink/Refinement.lean` (fn-85 .6) — the stuttering simulation
- `model/Temporal/Feature/Nexus/Caller/Model.lean` (fn-85 .10; until then `Temporal/Feature/Nexus/Tests/Machines.lean:105-123`) — `nexusProduct`
- `model/ModelLint/ImportGraph.lean:120` — `implementationLinkConsumers`

**Optional:**
- `.plans/UMPIRE4_SPEC.md:130-132,170-171` — SEM-08 and MOD-10

### Key context
- The spec's decision: `Temporal.System.Nexus` stays; deleting it would amend SEM-08 and MOD-04. Re-anchoring keeps that decision and unblocks the deletions.

## Acceptance
- [ ] the Implementation Link imports no module on the deletion list; `checkImplementationLink` reports `.complete` with the same obligations discharged; the terminal-closure check still runs over the product machine's `ends:`
- [ ] the link's evidence tests pin the Caller Model's Queries and pass; any element with no product counterpart is a listed Known Gap in required coverage
- [ ] `lake build TemporalModelTests Temporal.System.Nexus.Tests` green; `make lint-model` green with the single MOD-10 exception unchanged


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

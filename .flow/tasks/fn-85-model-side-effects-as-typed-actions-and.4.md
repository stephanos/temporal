---
satisfies: [R1, R4]
---
# fn-85-model-side-effects-as-typed-actions-and.4 Entity instances and structured machine state in the Contract

## Description
Let a Model hold several instances of each entity bounded by Limits, with Search, admission and path selection over every instance's state (R1), and carry per-instance state fields in the Contract instead of one state value plus facts: the correlated transition records the fields the machine keeps, keyed by the entity's key (the structured machine state fn-87 left to this spec). Interleavings across instances are paths (R4).

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/correlated.proto` (a repeated named-value field record on the correlated transition and initial state, additive on fn-87's shapes), `api/testpilot/v1/*`, `model/Testpilot/Authoring.lean` and `Correlated.lean` (structured state in the Lean interpreter), `model/Umpire/Case/Correlated.lean` and `Projection/*.lean` (lower machine fields, not one atom), `model/Umpire/Property.lean` and `Property/Evaluate.lean` (`PropertyTraceField.state` per field; `naturalAtMost` reads a `count` field), `model/Umpire/Search.lean` (instances as candidate setups; interleavings), `model/Umpire/Query.lean` (`FiniteDomain` fingerprints over instances), `common/testing/testpilot/internal/verification/{correlated,correlated_prepare}.go` (structured state), `common/testing/testpilot/testdata/case-runtime-conformance/correlated.json` (regenerated), conformance corpus (a correlated case over two instances), `model/Umpire/Case/Tests/*.lean`
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Umpire/Property.lean, model/Umpire/Property/**, model/Umpire/Search.lean, model/Umpire/Query.lean, common/testing/testpilot/**]

### Approach
- The protocol-migration oracle is retired in `.1`, so no declared mapping step is needed here; the
  conformance `expected.json` pins are the Verdict net.
- Protocol: follow fn-87's extension checklist (`common/testing/testpilot/README.md`) for the new correlated shape; additive fields, no compatibility shim, fixtures regenerate through their generators, a Driver conformance case per changed message.
- State: an instance's state is a record of named fields; the Contract's correlated transition carries `prior` and `next` field records and the projection rule matches on fields, so `attempts` compares as a number and `phase` as an enum without parsing one atom.
- Instances: the fixed-width tuple from task .3 becomes the setup shape; `Query.FiniteDomain.canonicalRoleAssignments` enumerates instance assignments; a bound of zero rejects; exceeding a bound reports `limitReached`.
- Interleavings: Search treats every enabled step of every instance as a candidate; the exact-sequence Scenario still pins one order.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/testpilot/v1/correlated.proto` (post-fn-87) — the transition and initial-state messages
- `model/Umpire/Case/Correlated.lean:28-216` and `model/Testpilot/Correlated.lean:29-45,211-352`
- `model/Umpire/Property.lean:303-361` and `Property/Evaluate.lean:60-120` — trace fields and numeric predicates
- `model/Umpire/Search.lean:375,760-795` — setups and root traces
- `common/testing/testpilot/internal/verification/correlated_prepare.go`

**Optional:**
- `common/testing/testpilot/README.md` — the extension section fn-87 R7 wrote

### Key context
- fn-87's boundary table assigns "correlated transitions over structured machine state" to this spec; this is that task.
- Memory: check unbounded Lean numbers before protobuf narrowing (`count` crosses into a fixed-width field).

### Deferred here from task .14, 2026-09-18

`DESIGN.md` writes an instance's creation as a row from `none`, taken by the action that `creates:`
the entity. That is a different shape from a step over an existing state -- its function takes no
prior state, and its successors are the machine's *initial* states rather than transition rows -- so
`machine` takes a `starts:` key naming those states directly, as the `model` command did. Deciding
whether a creating action's step function produces the initial states, and whether that action
belongs in the machine's enumerated Action domain at all, needs the instance model this task owns.

## Acceptance
- [x] the correlated Contract carries per-instance state fields; the Lean interpreter and the Go evaluator agree on `correlated.json` and on a new two-instance conformance case
- [x] a Model with two operation instances admits, searches interleavings, and a Query selecting one path per instance produces a Case; an instance bound of zero rejects in place; an exhausted bound reports `limitReached`
- [x] the async-Nexus fixture regenerates with only the structured-state diff, listed in the receipt; `make umpire-check-regression` exit 0


## Done summary

Delivered in two halves. The first (d9e769a, d00faa9, in the previous session) put a machine's
state fields on the Model and on the wire and made both evaluators read a STATE condition as the
state and every field it holds. This half makes the fields readable on the Model side too, puts a
structured two-operation Target into the conformance corpus, and lets a Scenario run over several
instances of one entity.

### State fields on the Model side

The Model-side Property evaluator reads a state as itself and every field it holds -- the same pair
the runtime's correlated evaluator reads off the wire -- so a pattern names either the state's
definition or a field's. `CheckedModel` carries each state's fields, `FiniteModelIdentity` says
what they are, and `declareModel` provides one meaning per field beside the states, because the
evaluator admits a value only through a meaning of the capability the Property requires. That
meaning is what moved every Property-over-a-machine fingerprint once (the `.15` pins now carry the
moved values and say why), and it is what lets the correlated certificate read a field-naming
rule on this side exactly as the portable step condition does on that one: `CorrelatedProofs`
builds its row input with the row's own fields.

### The structured conformance family

`correlated.json` gains four scenarios over a second Target -- `open`/`done` phase and an attempt
count, each a field on the wire -- whose one rule reads the `attempts` field after a `fault`:
one fault satisfies it, a second (`faultAgain`, the recorder saying which fault this was) violates
it, two operations that each fault once are two operations, and an interleaved second fault on one
of them is a violation. The Lean `#guard` and the Go facade replay answer all four alike. Two Go
tests raised their per-event work ceiling for it and one its fixture count; a projection rule's
confirmed steps turned out to be a sequence, not alternatives, which is recorded on the spec.

### Instances

`instances: 2` on a `scenario`, with each action naming the instance that takes it (`awaitStart 2`),
runs the Search over the product of that many copies of the machine (`Umpire.Command.Instances`):
a state per slot, an action per (slot, action), rows that step one slot by the machine's own row,
the machine's own canonical-table law as the product's authority, and catalogs emitted in the order
Search admits. Each slot is a state field of the product state, which is what lets one instance's
Property be read over the product as the same claim per instance on the acting slot. The Producer
reads the first instance back -- the machine as declared, its own sequence, the projection of the
selected path -- with every instance's actions as the Program's path (`Realizable.program`), so a
Case follows one operation through the machine while its Program performs both. Pinned on the
success Model: a two-instance Query is found with both interleavings as paths, its Property is two
clauses per instance, the realizable view is one instance's two steps, a Case is produced, and a
search budget of one reports the bound stopped it.

Rejections, each where it is written: an instance count of zero, a count above nine (the product's
keys number instances by one digit, and the canonical order Search admits is the order of those
keys), a product past the enumeration bound (with both factors and the bound), an unnumbered action
under `instances:`, a numbered action without it, an instance outside the count, and -- at the
Query, since the Scenario alone is admissible -- instances that perform different sequences,
because a Case follows each operation through one sequence.

### The question deferred from `.14`

A creating action is a step from a start phase: `.16` already wrote the design's `none + schedule`
row as a step from `unscheduled`, so `schedule` belongs to the machine's Action domain like any
other and `starts:` names the phase. Nothing separate is needed for instances: every slot begins
in a start state and the creating action's step is the one that leaves it.

### The async-Nexus fixture diff

Eight lines, all fingerprints: the Target's, the projection's (three copies) and the Property's
(four copies, in the definitions and the two correlated rule bindings), each moved by the field
meaning. No Program, Contract or provenance row otherwise changed.

### Gates

`lake build` green (616 jobs); `make umpire-check-testpilot-authoring`, `umpire-check-case-runtime-conformance`
and `umpire-check-goldens` exit 0; `go test -tags test_dep ./common/testing/testpilot/...` green;
`LEAN_NUM_THREADS=1 make lint-model` at the 163 baseline. `make umpire-check-regression` in full is
run at the next task boundary that touches live tests; this task changed no live test's bytes
beyond the fixture's fingerprints.

Self-review: no second backend is installed in this cloud session, so this owes a cross-model
re-review before the completion review, as the tasks before it do.

### Re-review fix, 2026-09-19

The cross-model re-review found the per-slot lift of a transition claim unsound: a group triggered
by one slot's prior state has no clause naming which instance acts, so another instance's step
leaves the slot where it was and violates a claim that its next state differs (`startedThenSucceeds`
over `twoOperations` was found unsatisfiable and verified violated on a trace on which every step
out of `started` does reach `succeeded`). `checkInstances` now rejects a Property with a
prior-state group before searching, at the Scenario, and `twoTransitions` in the success tests
pins the message; a sound lift waits on a clause that can conjoin a slot's prior state with the
acting instance.

## Evidence
- Commits: d9e769a, d00faa9, 482f43f
- Tests: cd model && lake build; make umpire-check-testpilot-authoring; make umpire-check-case-runtime-conformance; make umpire-check-goldens; go test -tags test_dep ./common/testing/testpilot/...; LEAN_NUM_THREADS=1 make lint-model
- PRs:

## Goal & Context
<!-- scope: business -->

Status: implementation specification derived from the 2026-09-08 assessment of Umpire and
Testpilot against [UMPIRE4_VISION](../../.plans/UMPIRE4_VISION.md).
[UMPIRE4_SPEC](../../.plans/UMPIRE4_SPEC.md) remains normative.

The assessment found that the substrate the vision needs exists and is rigorous, while the
product-facing seams the vision names are thin. This spec closes the seams that block a Temporal
engineer from writing one checked model and running it as a regression:

1. The shipped Nexus3 Producer hand-writes its Program and its four-state monitor, then rejects
   any checked Property that is not clause-for-clause equal to one expected value. A model edit
   produces a lowering error instead of a different Case. The proof-carrying scoped lowering in
   `Umpire.Case.Scoped` is the only path where wire bytes provably mean the checked model, and no
   shipped Case uses it.
2. The five-block Nexus3 command syntax rejects every identifier spelling except one whitelist,
   and its `query ... all ...` form always errors. A newcomer who renames an Action or adds a
   transition gets a macro error with no path forward.
3. Classic Contract rules bound liveness only by an elapsed-milliseconds horizon that the recorder
   derives from the test host's clock. The scoped capability already counts admitted operation
   transitions and never ticks on elapsed time. A slow CI runner can turn a healthy target into a
   violated verdict on the classic path.
4. Fault intents exist in `Umpire.Space` and `Umpire.Artifact` and are carefully labeled as
   intent. No opcode, capability, Driver method, or Run Event kind can realize one. The only
   negative live test works by not creating a Nexus endpoint.
5. Running one Case from Go costs 60 to 80 lines, most of it a `ProfileSpec` that restates role
   kinds, gRPC method allowlists, reservation carrier shapes, and every capability the Case
   already implies. A mismatch surfaces only at `Prepare`. No helper derives a Profile and no
   runner helper exists.
6. The execution runtime discards the recorder close error and returns `nil` from `Run`.

Sequencing: fn-77 task .8 landed the whole-Case coverage modules R1 consumes. fn-77 tasks .9 and
.10 modify the Nexus3 Producer, the async-nexus fixture bytes, and the live test, so R1 and R5
serialize behind fn-77.10 to avoid a byte conflict on regenerated fixtures. R3, R4, and R6 do not
touch those files and may start now. R2 and R8 follow R1 so both edit the Nexus3 test module in
one order.

## Architecture & Data Models
<!-- scope: technical -->

### Ownership

| Owner | Responsibility |
| --- | --- |
| `Umpire.Case.Scoped`, `Umpire.Case.Coverage`, `Umpire.Case.Compiler` | The only checked lowering from a checked Property, Behavior, Query, and selected witness to a Testpilot Contract. The proof-carrying `Lowered` value remains the correspondence certificate. |
| `Temporal.Feature.Nexus3.Testpilot` | Program realization keyed by Nexus3 Action identity and evidence projection declarations. It supplies Program and projection inputs to the Umpire lowering. It authors no monitor rule and compares no Property by equality. |
| `Temporal.Feature.Nexus3.Syntax` and `Authoring` | Command syntax that elaborates any enum-like finite lifecycle within declared bounds into `Umpire.FiniteTable` and the existing `PropertySpec`, `ExactSequenceSpec`, `QueryLimitSpec`, and `Authoring.check` owners. |
| Testpilot protobuf closure | Wire authority for the event-count horizon, the fault instruction, the fault capability, the fault Run Event kind, and its declared event fields. |
| Testpilot internal verification | Ticks the event-count horizon through one helper shared by online and offline evaluation. |
| Testpilot internal execution | Dispatches the fault instruction through the Driver, records the fault Run Event, and returns the recorder close error. |
| Temporal worker Driver | Realizes the first fault kind by stopping and resuming the SDK worker of a dedicated, non-pooled worker group for one activation queue. |
| Temporal composite Driver package | Derives a minimal `ProfileSpec` from a Case, a Catalog, and an environment description. |
| Functional fixture package under tests | One runner helper that decodes, derives, prepares, and runs a Case fixture against a live cluster, and the fault Case fixture with its live test. |

### R1 general lowering

`produceCompletionCase` currently takes the checked values, throws on any scoped clause, then
compares each input to a fixed expected model. Replace it with `produce` over one
`Authoring.CheckedModel`:

- Program realization keyed by Action identity. Each Nexus3 Action maps to the Program nodes
  that request it. The map is data in the Producer.
- Evidence projection declarations keyed by modeled Fact and result field, consumed by
  `Umpire.Observation.Projection.Coverage`. The three success clauses (resulting state, outcome,
  fact) are expressed as scoped trigger and response predicates through the existing
  `bounded_response%` clause form.
- One call to `Umpire.Case.Scoped.lower` with coverage passed explicitly, never defaulted, then
  `Umpire.Case.Compiler.compile` with the coverage request populated from the checked Property's
  field operands.

The hand-written success rule and the clause-shape match are deleted. Of the protections the
equality gate provided, witness absence stays a `LoweringError`, an unexpressible clause stays a
`LoweringError` naming the clause, and a changed Target, Behavior, Query, or witness becomes
different Case bytes rather than an error. The async-nexus fixture and all six conformance
expectation trees are regenerated because one gate diffs both trees. The Case's liveness bound
comes from the scoped operation-transition clock; it declares no classic horizon.

### R2 general syntax and R8 verify form

The five command macros keep their surface spelling. Their bodies change from a spelling
whitelist to elaboration over the parsed identifiers:

- `model` accepts any role name, any enum-like state, action, outcome, and fact inductive, any
  initial and terminal lists, and up to 256 transitions. It elaborates into a `FiniteTable` value
  and derives the ordered domains and enumerators from the inductive constructors. Enumeration
  completeness, domain membership, and Action executability are discharged by `decide` or `rfl`.
  A constructor with arguments, an unknown constructor, a duplicate `before + action` pair, or a
  terminal state unreachable from every initial state over the transition table is a located
  elaboration error. The tested scale is at most 16 states; 256 is an elaboration bound.
- `property` accepts any number of `require` clauses over `resultingState`, `outcome`, `fact`,
  and the field comparison forms fn-77 introduced.
- `behavior` accepts `exactly` and the existing `Umpire.Behavior` occurrence bounds.
- `limits` accepts any positive numbers.
- `query ... all ...` (R8) elaborates through `QueryForm.verify`. The Producer rejects a verify
  Query with a witness-absent `LoweringError` because a Case realizes one selected trace.

The elaborators reuse `Authoring.check` and report failures through the located diagnostic path
the `property%`, `behavior%`, and `query%` elaborators already own.

### R3 event-count horizon

`ContractHorizonDefinition` gains a plain `int64 rule_events` field beside the existing
`elapsed_milliseconds`. A protobuf `oneof` is avoided because it would change the generated Lean
mirror from a scalar to an inductive at every horizon site. Admission requires exactly one of the
two bounds to be positive.

`rule_events` counts every Run Event the rule evaluated while in a nonterminal state, reset to
zero on each transition into a new state. Counting continues while execution is incomplete, but
the expiry conclusion is suppressed there, matching the recorded evaluator precedence. Counting
stops once the rule reaches a terminal state. One helper owns the counter and both
`Evaluator.Observe` and `PreparedContract.Evaluate` call it. `elapsed_milliseconds` remains
admitted and is documented as host-clock dependent.

### R4 one fault

- `Instruction` gains `InjectFault { string role_id; FaultKind kind; }` with
  `FAULT_KIND_WORKER_STOP` and `FAULT_KIND_WORKER_RESUME`. Context is `CONTROLLER` only.
  `role_id` must name a `ROLE_KIND_TASK_QUEUE` role; that role's resource binding identifies the
  queue.
- `Capability` gains `InjectFault` as the next value of the existing `uint8` iota. The Opcode
  enum and the instruction-to-opcode switch gain the matching entries, and a test asserts the
  three lists stay aligned.
- `Session` gains `InjectFault(ctx, Coordinate, roleID, kind) (EffectHandle, error)`.
- `RunEventKind` gains `FAULT_INJECTED`, and every runtime and verifier range check that treats
  the diagnostic kind as the maximum admits it. `RunEventField` (declared in the expression
  schema) gains `FAULT_ROLE_ID` and `FAULT_KIND`; the IR reference range check and the Contract
  event-field scope admit them so a Contract expression can inspect them under EVD-13. The Run
  Event carries the two values only for fault events.
- A prepared Program containing `InjectFault` opens a dedicated worker group keyed by the Run ID
  instead of a pooled group, so a stop never affects another Run. The Driver realizes
  `WORKER_STOP` by stopping that group's SDK worker and suppressing its fatal-failure callback
  for the stop window, and `WORKER_RESUME` by re-registering with the same structural signature.
  Both are bounded by the instruction timeout. Cleanup resumes a stopped worker before releasing
  the group. Tasks queued during the stop window wait in matching and dispatch after resume.
- `Umpire.Space.FaultIntentDeclaration.lower` produces the `InjectFault` instruction definition
  placed before the occurrence it names. `Umpire.Exploration.Coverage` keeps its wording that a
  requested fault is intent until the Run carries a `FAULT_INJECTED` event for it.
- The acceptance Case is a Lean-authored functional fixture under the tests fixture package. It
  is a Producer-neutral Case like the existing system-info fixture, not a seventh conformance
  class, so EVD-18 stays true. Its classic rule declares a `rule_events` horizon.

### R5 Profile derivation and runner

- `temporal.DeriveProfile(source, catalog, env) (ProfileSpec, error)` returns the minimal policy
  the Case implies. Role kinds, method allowlists, carrier shapes, capabilities, and environment
  bindings come from the Case and the Catalog. A carrier shape's maximum count is the number of
  reserving nodes per entrypoint context in the Case. Identity comes from `env.Identity`. Anything
  the Case references that the Catalog does not know is an error. The result is a value the
  caller may tighten before `Prepare`.
- The hand-written async-nexus Profile is retained as the derivation oracle. A test under the
  fixture package asserts the derived Profile equals it. The live test uses the derived value.
- The fixture package gains a two-layer runner. `BindCase` takes a decoded Case and an explicit
  `CaseBinding` (identity, namespace, task queue, Nexus endpoint, and whether to create the
  endpoint), derives the Profile, provisions, builds the shared Driver, prepares, and returns a
  binding that can run. `RunCase(t, env, name, binding) (*Run, *Verdict)` loads the named fixture,
  binds with endpoint creation, runs once, and fails the test on a non-nil error. Tests that vary
  bindings or omit the endpoint use `BindCase` directly. Both are test helpers outside the public
  facade, preserving MOD-12.

### R6 close error

`Run` returns the recorder close error alongside the immutable Run and Verdict. Existing callers
already check the error, and a successful close is unchanged, so no conformance expectation
changes.

## API Contracts
<!-- scope: technical -->

Wire changes, all additive under ART-04:

```proto
message ContractHorizonDefinition {
  int64 elapsed_milliseconds = 1;
  string violation_state_id = 2;
  int64 rule_events = 3;
}

message InjectFault {
  string role_id = 1;
  FaultKind kind = 2;
}

enum FaultKind {
  FAULT_KIND_UNSPECIFIED = 0;
  FAULT_KIND_WORKER_STOP = 1;
  FAULT_KIND_WORKER_RESUME = 2;
}

// run.proto
enum RunEventKind { RUN_EVENT_KIND_FAULT_INJECTED = 11; }
// expression.proto (existing enum, last value RUN_ID = 9)
enum RunEventField { RUN_EVENT_FIELD_FAULT_ROLE_ID = 10; RUN_EVENT_FIELD_FAULT_KIND = 11; }
```

Go additions:

```go
// common/testing/testpilot
const InjectFault Capability = iota + 1 // appended to the existing list

type Session interface {
    // existing methods unchanged
    InjectFault(ctx context.Context, at Coordinate, roleID string, kind testpilotspb.FaultKind) (EffectHandle, error)
}

// common/testing/testpilot/temporal
type Environment struct {
    Identity      string
    Namespace     string
    TaskQueue     string
    NexusEndpoint string
}
func DeriveProfile(source *testpilotspb.Case, catalog *testpilot.Catalog, env Environment) (testpilot.ProfileSpec, error)
```

Lean additions:

```lean
-- Temporal.Feature.Nexus3.Testpilot
def produce (checked : Authoring.CheckedModel lifecycle) : Except LoweringError Case

-- Testpilot.Authoring
def Monitor.horizonEvents (ruleEvents : Int64) (violationStateId : String) : ContractHorizonDefinition
def Program.injectFault (roleId : String) (kind : FaultKind) : Instruction

-- Umpire.Space
def FaultIntentDeclaration.lower : FaultIntentDeclaration → Except LoweringError InstructionDefinition
```

## Edge Cases & Constraints
<!-- scope: technical -->

- A checked Property whose clauses the lowering cannot express rejects with a `LoweringError`
  naming the clause. Known Gaps cannot waive it (fn-77 R9 stands) and a test proves it.
- A horizon with both bounds positive, both zero, or `rule_events` negative rejects at `Prepare`.
  A horizon reached on the same event that would satisfy the rule resolves as expiry, because
  expiry runs before transitions.
- `InjectFault` on a role that is not a task-queue role rejects at `Prepare` as `malformed`. A
  Profile without the `InjectFault` capability rejects as `unsupported`. `WORKER_RESUME` without a
  prior `WORKER_STOP` in the same Run is a Driver invariant failure recorded as a diagnostic; the
  Verdict is unaffected.
- A stopped worker that fails to resume within cleanup bounds sets cleanup status `failed`. The
  Verdict is unaffected (QLF-05).
- `DeriveProfile` never widens beyond what the Case references. A Case with no worker roles yields
  no worker policy and no reservation carriers.
- Regenerated fixtures compare byte-for-byte under the conformance gate. `make
  umpire-check-live-tests` must still pass; fn-81 retired its pinned expected-failure list, so the
  gate now requires an empty failure set across the `^TestTestpilot` selector rather than an
  unchanged list.

## Quick commands

```bash
cd model && lake build Temporal TemporalModelTests UmpireTests TestpilotTests
make lint-model
make umpire-check-case-runtime-conformance
CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/...
go test -tags 'test_dep integration' ./tests -run 'TestTestpilot'
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `Temporal.Feature.Nexus3.Testpilot.produce` builds the async-nexus Case through
  `Umpire.Case.Scoped.lower` and `Umpire.Case.Compiler.compile` with no hand-written monitor rule
  and no equality comparison against an expected Property. Editing one `require` clause in the
  Nexus3 model changes the regenerated Contract bytes and no Lean source file other than the
  model file. Errors: an unexpressible clause rejects with a `LoweringError` naming it; a Known
  Gap does not admit it; a verify-form Query rejects as witness-absent; a Property edit that
  removes coverage for a projected field rejects at `compile` before any Driver I/O.
- **R2:** The `model`, `property`, `behavior`, and `limits` commands elaborate a second lifecycle
  shaped like the Nexus2 cancellation race, with a different role name, four states, three
  transitions, and a two-clause Property, and a witness Query over it checks. Errors: a
  constructor with arguments, an unknown constructor, a duplicate transition source, an
  unreachable terminal, and a transition count over 256 each produce a located elaboration error
  asserted with `#guard_msgs`; the three whitelist error strings are deleted.
- **R3:** A classic Contract rule declares a `rule_events` horizon. One shared helper expires it
  after exactly that many evaluated events since the rule's last transition, online and offline
  agree on the same event sequence, and a checked-in Lean-authored Case declares it. Errors: both
  bounds positive, both zero, or a negative count rejects at `Prepare`; expiry on the satisfying
  event resolves as expiry; counting continues under incompleteness while expiry is suppressed.
- **R4:** A Lean-authored functional Case issues `InjectFault WORKER_STOP` on the task-queue role
  before its start-workflow instruction and `WORKER_RESUME` after a bounded wait, runs live
  through the shared Driver in a dedicated worker group, records one `FAULT_INJECTED` event per
  instruction with `role_id` and `kind` fields, its Contract references both events in order, and
  its correlated success rule is satisfied. `FaultIntentDeclaration.lower` produces the same
  instruction definition. Errors: a missing `InjectFault` capability rejects at `Prepare` as
  `unsupported`; a non-task-queue role rejects as `malformed`; a resume that times out sets
  cleanup `failed` and leaves the Verdict unchanged; a concurrent Run without the instruction is
  unaffected by the stop.
- **R5:** `temporal.DeriveProfile` equals the hand-written async-nexus Profile, identity included,
  for the async-nexus Case, and `BindCase` plus `RunCase` replace the manual sequence in the live
  tests, including the two-environment and missing-endpoint cases, with no change in asserted
  Verdicts. Errors: an unknown method or role kind in the Case is a `DeriveProfile` error; the
  derived Profile never contains a capability, method, or carrier the Case does not use; a
  binding that omits endpoint creation still yields the incomplete and inconclusive result.
- **R6:** `PreparedCase.Run` returns a non-nil error when the recorder close fails, with the Run
  and Verdict still returned and unchanged, and a test drives that path. Errors: no error
  surface beyond the returned error; conformance expectations are unchanged.
- **R7:** `make lint-model`, `make umpire-check-regression`, and the Go packages under the
  Testpilot facade and the fixture package pass. Theorem axiom inventories match the approved
  baseline. New spec rules for Driver-realized faults, horizon units, and macro-derived finite
  domains are drafted under new IDs for human approval under GOV-02, and the roadmap gains an
  fn-80 entry. Errors: a failed gate blocks completion; a rule edit without approval blocks
  completion.
- **R8:** The `query ... all ...` command elaborates through `QueryForm.verify` and checks the R2
  lifecycle. Errors: the Producer rejects a verify Query as witness-absent; a verify Query over an
  unsatisfiable Behavior reports `unsatisfiable`, never a passing check.

## Early proof point

Task fn-80-close-the-model-to-case-seam-and-harden.4 validates the core approach: the shipped
success Property lowers through the proof-carrying scoped path and the regenerated Case still
satisfies live. If it fails, re-evaluate whether the scoped clause form can carry state, outcome,
and fact predicates before continuing with R2 and R8.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | General checked lowering replaces the equality gate | .4 | — |
| R2 | Generalized model, property, behavior, limits syntax | .5 | — |
| R3 | Event-count horizon for classic rules | .1, .8 | — |
| R4 | One worker fault end to end | .2, .3, .8 | — |
| R5 | Profile derivation and runner | .7 | — |
| R6 | Run returns the recorder close error | .2 | — |
| R7 | Gates, axiom baseline, spec rules, roadmap | .9 | — |
| R8 | `query ... all ...` verify form | .6 | — |

## Boundaries
<!-- scope: business -->

- No canary Profile, read-only mode, or production authorization. fn-70 and fn-29 own those.
- No Nexus cancellation lowering. fn-79 owns it and remains deferred.
- No white-box observation source. Observations remain gRPC response projections.
- No second fault kind. Server-side, network, and persistence faults need their own design.
- No rule-level Run Event filter. `rule_events` counts evaluated events.
- No removal or deprecation of `elapsed_milliseconds`.
- No new `RunDiagnosticKind`.
- No change to worker pooling for Cases without `InjectFault`.
- No Case checksum field. Fixture determinism stays with the generator and `make` gates.
- No activity entrypoint support. The dead activity entrypoint kind is not touched.
- No changes to the system-info or conformance Producers beyond regenerated bytes. They remain
  Producer-neutral non-model Cases under SEM-18.
- No CLI `run` verb. The runner lives in the Go test package.
- No second syntax lifecycle beyond the one R2 names.

## Decision Context
<!-- scope: both -->

The assessment's strongest finding was that one proven lowering path exists and the shipped Case
bypasses it. Keeping the hand-written monitor, for example by adding a second expected Property
to the equality gate, leaves the vision's "model verifies regression" claim unsupported. Routing
the success Property through the scoped path costs a fixture regeneration and gains a
correspondence proof for the shipped Case.

Generalizing the macros was chosen over deleting them. The syntax is the only surface that reads
well for a newcomer, fn-67 is already open to refine its documentation, and the elaboration
target already exists in `FiniteTable` and `Authoring.check`. Rejected as overkill: a general
Lean deriving handler for arbitrary inductives; enum-like constructors cover every lifecycle in
the tree.

The event-count horizon copies the scoped clock's design rather than introducing a Driver clock,
because EVD-07 forbids conclusions that rest on synchronized wall clocks and the scoped path has
already proven that admitted transitions are a sufficient tick. A plain field was chosen over a
protobuf `oneof` to keep the generated Lean mirror scalar.

Worker stop and resume was chosen as the first fault because the worker Driver already owns
registration and stop timeouts, it needs no server-side API, and it exercises the reservation
ledger's release path under a real outage. A dedicated worker group per fault-carrying Run was
chosen over a Prepare-time rejection of pooled workers because it eliminates cross-Run
interference structurally.

Profile derivation is a convenience for local and CI callers. The Profile remains an
authorization snapshot under QLF-01, so the derived value is returned for review rather than
applied silently. Canary callers keep hand-authored Profiles.

The declined ledger entry on generated API drift verification is unaffected: this spec adds no
generated-API drift gate and no CI workflow beyond the existing regression gates.


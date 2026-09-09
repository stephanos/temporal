## Goal & Context
<!-- scope: business -->

Status: implementation specification derived from the 2026-09-08 assessment of Umpire and
Testpilot against [UMPIRE4_VISION](../../.plans/UMPIRE4_VISION.md).
[UMPIRE4_SPEC](../../.plans/UMPIRE4_SPEC.md) remains normative.

The assessment found that the substrate the vision needs exists and is rigorous, while the
product-facing seams the vision names are thin. This spec closes the five seams that block a
Temporal engineer from writing one checked model and running it as a regression:

1. The shipped Nexus3 Producer in `Temporal.Feature.Nexus3.Testpilot` hand-writes its Program and
   its four-state monitor, then rejects any checked Property that is not clause-for-clause equal
   to one expected value. A model edit produces a `LoweringError` instead of a different Case.
   The proof-carrying scoped lowering in `Umpire.Case.Scoped` is the only path where wire bytes
   provably mean the checked model, and no shipped Case uses it.
2. The five-block Nexus3 command syntax in `Temporal.Feature.Nexus3.Syntax` rejects every
   identifier spelling except one whitelist, and its `query ... all ...` form always errors. A
   newcomer who renames an Action or adds a transition gets a macro error with no path forward.
3. Classic Contract rules bound liveness only by `ContractHorizonDefinition.elapsed_milliseconds`,
   which the recorder derives from the test host's clock. The scoped path already counts admitted
   operation transitions and never ticks on elapsed time. A slow CI runner can turn a healthy
   target into a violated verdict on the classic path.
4. Fault intents exist in `Umpire.Space` and `Umpire.Artifact` and are carefully labeled as
   intent. No opcode, capability, Driver method, or Run Event kind can realize one. The only
   negative live test works by not creating a Nexus endpoint.
5. Running one Case from Go costs 60 to 80 lines, most of it a `ProfileSpec` that restates role
   kinds, gRPC method allowlists, reservation carrier shapes, and every capability the Case
   already implies. A mismatch surfaces only at `Prepare`. No helper derives a Profile and no
   runner helper exists.

A sixth, smaller finding is included because it is a one-line correctness hole in the same
runtime: `internal/execution/runtime.go` discards the recorder close error and returns `nil`.

Sequencing: fn-77 task .8 is in progress and touches `model/Umpire/Case/**` and
`model/Umpire/Observation/Projection/**`. R1 starts after fn-77.8 lands. R3, R4, R5, and R6 do
not depend on fn-77 and may proceed in parallel. R2 depends on R1 so the generalized syntax
elaborates into the lowering path rather than the equality gate.

## Architecture & Data Models
<!-- scope: technical -->

### Ownership

| Owner | Responsibility |
| --- | --- |
| `Umpire.Case.Scoped`, `Umpire.Case.Coverage`, `Umpire.Case.Compiler` | The only checked lowering from a `CheckedProperty`, `CheckedBehavior`, `CheckedQuery`, and selected witness to a Testpilot Contract. Proof-carrying `Lowered` values remain the correspondence certificate. |
| `Temporal.Feature.Nexus3.Testpilot` | Program realization for Nexus3 Actions and evidence projection declarations. It supplies Program and projection inputs to the Umpire lowering and no longer authors monitor rules or compares Properties by equality. |
| `Temporal.Feature.Nexus3.Syntax` and `Authoring` | Command syntax that elaborates any finite lifecycle within declared bounds into `Umpire.FiniteTable` and the existing `PropertySpec`, `ExactSequenceSpec`, `QueryLimitSpec`, and `Authoring.check` owners. |
| `proto/internal/temporal/server/api/testpilot/v1` | Wire authority for the event-count horizon, the fault instruction, the fault capability, and the fault Run Event kind. |
| `common/testing/testpilot/internal/verification` | Ticks the event-count horizon identically online and offline. |
| `common/testing/testpilot/internal/execution` | Dispatches the fault instruction through the Driver, records the fault Run Event, returns the recorder close error. |
| `common/testing/testpilot/temporal/worker` | Realizes the first fault kind by stopping and resuming a registered worker for one activation queue. |
| `common/testing/testpilot/temporal` | Derives a minimal `ProfileSpec` from a Case plus an environment description. |
| `tests/testcore/testpilot` | One runner helper that decodes, derives, prepares, and runs a Case fixture against a live cluster. |

### R1 general lowering

`Temporal.Feature.Nexus3.Testpilot.produceCompletionCase` currently takes the checked values, throws
on any scoped clause, then compares each input to a fixed expected model. Replace the body with:

- Program realization keyed by Action identity. Each Nexus3 Action maps to the Program nodes that
  request it. The map is data in the Producer, not a hand-written Program per Case.
- Evidence projection declarations keyed by modeled Fact and result field, consumed by
  `Umpire.Observation.Projection.Coverage`.
- A single call to `Umpire.Case.Scoped.lower` followed by `Umpire.Case.Compiler.compile` with the
  coverage request populated from the checked Property's field operands.

The hand-written `successRule` and `supportsSuccessProperty` are deleted. The three-clause success
Property is expressed as scoped clauses so the same path lowers it. The existing async-nexus fixture
is regenerated. Its Program bytes may change. Its Contract must now carry `contract.scoped` and the
regenerated `expected.json` records the Verdict shape the live test asserts.

### R2 general syntax

The five command macros keep their surface spelling. Their bodies change from a spelling whitelist
to elaboration over the parsed identifiers:

- `model` accepts any role name, any state, action, outcome, and fact inductive types, any initial
  and terminal lists, and any number of transitions up to a declared bound. It elaborates into a
  `FiniteTable` value plus the ordered domains and enumerators `AUT-08` requires, deriving them
  from the inductive constructors. A transition that names an unknown constructor, a duplicate
  `before + action` pair, or an unreachable terminal state is a located elaboration error.
- `property` accepts any number of `require` clauses over `resultingState`, `outcome`, `fact`, and
  the field comparison forms fn-77 introduced.
- `behavior` accepts `exactly` and the existing `Umpire.Behavior` occurrence bounds.
- `limits` accepts any positive numbers.
- `query` implements the `all` form through the existing `QueryForm.verify` path.

The elaborators reuse `Authoring.check` and report failures through the located diagnostic path
`property%`, `behavior%`, and `query%` already own.

### R3 event-count horizon

`ContractHorizonDefinition` gains a `oneof bound { int64 elapsed_milliseconds; int64 rule_events }`.
`rule_events` counts Run Events that pass the rule's `RunEventFilter` after the rule entered its
initial state. The evaluator applies expiry before transitions on every event, as `EVD-12` requires,
and both `Evaluator.Observe` and `PreparedContract.Evaluate` tick the same counter. The regenerated
async-nexus Case declares a `rule_events` horizon on its success rule. `elapsed_milliseconds` remains
admitted for existing fixtures and is documented as host-clock dependent.

### R4 one fault

- `Instruction` gains `InjectFault { string role_id; FaultKind kind; }` with
  `FAULT_KIND_WORKER_STOP` and `FAULT_KIND_WORKER_RESUME`. Context: `CONTROLLER` only.
- `Capability` gains `InjectFault`. A Profile that omits it rejects the instruction at `Prepare`.
- `Session` gains `InjectFault(ctx, Coordinate, roleID string, kind) (EffectHandle, error)`.
- `RunEventKind` gains `FAULT_INJECTED`, carrying `role_id` and `kind` as declared event fields a
  Contract may inspect.
- The worker Driver realizes `WORKER_STOP` by stopping the SDK worker registered for the role's
  task queue and `WORKER_RESUME` by re-registering it with the same structural signature. Both are
  bounded by `InstructionLimits.timeout_milliseconds`. Cleanup always resumes a stopped worker.
- `Umpire.Space.FaultIntentDeclaration` lowers to one `InjectFault` node placed before the
  occurrence it names. `Umpire.Exploration.Coverage` keeps its wording that a requested fault is
  intent until the Run carries a `FAULT_INJECTED` event for it.

### R5 Profile derivation and runner

- `temporal.DeriveProfile(source *Case, catalog *Catalog, env Environment) (ProfileSpec, error)`
  returns the minimal policy the Case implies. Every role kind, method allowlist, carrier shape,
  capability, and environment binding comes from the Case and the catalog. Anything the Case
  references that the catalog does not know is an error. The result is a value the caller may
  tighten before `Prepare`.
- `testpilotfixture.RunCase(t, env, name string) (*Run, *Verdict)` in `tests/testcore/testpilot`
  decodes the named fixture, derives the Profile, builds the shared Driver, prepares, and runs. It
  is a test helper and does not enter the public facade, preserving `MOD-12`.

### R6 close error

`Run` returns the recorder close error alongside the immutable Run and Verdict. The Run and Verdict
keep their `EVD-15` closure semantics.

## API Contracts
<!-- scope: technical -->

Wire changes, all additive under `ART-04`:

```proto
message ContractHorizonDefinition {
  oneof bound {
    int64 elapsed_milliseconds = 1;
    int64 rule_events = 3;
  }
  string violation_state_id = 2;
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

enum RunEventKind {
  // existing values unchanged
  RUN_EVENT_KIND_FAULT_INJECTED = <next>;
}
```

Go facade additions:

```go
// common/testing/testpilot
const InjectFault Capability = "inject-fault"

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

Lean surface additions:

```lean
-- Temporal.Feature.Nexus3.Testpilot
def produce (checked : Authoring.CheckedModel lifecycle) : Except LoweringError Case

-- Umpire.Space
def FaultIntentDeclaration.lower : FaultIntentDeclaration → Except LoweringError InstructionDefinition
```

## Edge Cases & Constraints
<!-- scope: technical -->

- A checked Property whose clauses the lowering cannot express rejects with a `LoweringError`
  naming the clause. Known Gaps cannot waive it (fn-77 R9 stands).
- `rule_events` of zero rejects at `Prepare`. A horizon reached on the same event that would satisfy
  the rule resolves as expiry, because expiry runs before transitions.
- `InjectFault` on a role that is not a worker role rejects at `Prepare`. `WORKER_RESUME` without a
  prior `WORKER_STOP` in the same Run is a Driver invariant failure recorded as a diagnostic.
- A stopped worker that fails to resume within cleanup bounds sets cleanup status `failed`. The
  Verdict is unaffected (`QLF-05`).
- `DeriveProfile` never widens beyond what the Case references. A Case with no worker roles yields
  no worker policy and no reservation carriers.
- The generalized `model` macro bounds transitions at `SpaceLimits.v1` scale (256) and reports the
  bound in its error.
- Regenerated fixtures compare byte-for-byte under `make umpire-check-case-runtime-conformance`.
  The `umpire-check-live-tests` expected-failure list must not gain entries.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `Temporal.Feature.Nexus3.Testpilot` produces the async-nexus Case through
  `Umpire.Case.Scoped.lower` and `Umpire.Case.Compiler.compile` with no hand-written monitor rule
  and no equality comparison against an expected Property. Editing one `require` clause in
  `Nexus3/Nexus.lean` changes the regenerated Contract bytes and no other source file. Errors: an
  unexpressible clause rejects with a `LoweringError` naming it; a Property edit that removes
  coverage for a projected field rejects at `compile` before any Driver I/O.
- **R2:** The `model`, `property`, `behavior`, `limits`, and `query` commands elaborate a second
  Nexus lifecycle with a different role name, four states, three transitions, and a two-clause
  Property, and the `query ... all ...` form verifies it. Errors: unknown constructor, duplicate
  transition source, unreachable terminal, and a transition count over the bound each produce a
  located elaboration error; no whitelist error string remains.
- **R3:** A classic Contract rule declares a `rule_events` horizon. The evaluator expires it after
  exactly that many filtered events, online and offline agree, and the regenerated async-nexus
  Case uses it. Errors: `rule_events` of zero rejects at `Prepare`; expiry on the satisfying event
  resolves as expiry.
- **R4:** A Case with an `InjectFault WORKER_STOP` before the Nexus handler activation, and a
  `WORKER_RESUME` after a bounded wait, runs live through the shared Driver, records one
  `FAULT_INJECTED` event per instruction, and its Contract references those events. A Space
  `FaultIntentDeclaration` lowers to that instruction. Errors: missing `InjectFault` capability
  rejects at `Prepare`; a non-worker role rejects at `Prepare`; a resume that times out sets
  cleanup `failed` and leaves the Verdict unchanged.
- **R5:** `temporal.DeriveProfile` reproduces the existing hand-written `AsyncNexusProfile` for the
  async-nexus Case, and `RunCase` replaces the manual sequence in
  `tests/testpilot_async_nexus_case_test.go` with no change in asserted Verdicts. Errors: an
  unknown method or role kind in the Case is a `DeriveProfile` error; the derived Profile never
  contains a capability the Case does not use.
- **R6:** `PreparedCase.Run` returns a non-nil error when the recorder close fails, with the Run
  and Verdict still returned and unchanged. Errors: no error surface beyond the returned error.
- **R7:** `make lint-model`, `make umpire-check-regression`, and the Go packages under
  `common/testing/testpilot/...` and `tests/testcore/testpilot/...` pass. Theorem axiom inventories
  match the approved baseline. Errors: a failed gate blocks completion.

## Boundaries
<!-- scope: business -->

- No canary Profile, read-only mode, or production authorization. fn-70 and fn-29 own those.
- No Nexus cancellation lowering. fn-79 owns it and remains deferred.
- No white-box observation source. Observations remain gRPC response projections.
- No second fault kind. Server-side, network, and persistence faults need their own design.
- No Case checksum field. Fixture determinism stays with the generator and `make` gates.
- No activity entrypoint support. The dead `ENTRYPOINT_KIND_ACTIVITY` path is not touched.
- No changes to `GetSystemInfo` or the conformance Producers. They remain Producer-neutral
  non-model Cases under `SEM-18`.
- No CLI `run` verb. The runner lives in the Go test package.

## Decision Context
<!-- scope: both -->

The assessment's strongest finding was that one proven lowering path exists and the shipped Case
bypasses it. Every alternative that keeps the hand-written monitor, such as adding a second
expected Property to the equality gate, leaves the vision's "model verifies regression" claim
unsupported. Routing the success Property through the scoped path costs a fixture regeneration
and gains a correspondence proof for the shipped Case.

Generalizing the macros was chosen over deleting them. The syntax is the only surface that reads
well for a newcomer, fn-67 is already open to refine it, and the elaboration target already
exists in `FiniteTable` and `Authoring.check`.

The event-count horizon copies the scoped clock's design rather than introducing a Driver clock,
because `EVD-07` forbids conclusions that rest on synchronized wall clocks and the scoped path has
already proven that admitted transitions are a sufficient tick.

Worker stop and resume was chosen as the first fault because the worker Driver already owns
registration and stop timeouts, it needs no server-side API, and it exercises the reservation
ledger's release path under a real outage.

Profile derivation is a convenience for local and CI callers. The Profile remains an authorization
snapshot under `QLF-01`, so the derived value is returned for review rather than applied silently.
Canary callers keep hand-authored Profiles.

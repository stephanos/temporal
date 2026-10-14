# Portable evaluation contract and disposable-cluster qualification

## Umpire4 architecture reconciliation

Fn-28 proves that one model-produced Test can execute and reach a trustworthy local verdict in a
process that has no Lean runtime. Lean remains the sole source of behavioral meaning: it lowers the
selected Test, Observation, Implementation Link, and Property into one closed per-test evaluation
contract. Go admits and executes that contract through a fixed, versioned interpreter; it does not
invent behavior or make whole-model claims.

The proof uses one tagged Go integration test and `testcore.NewEnv` to supply a disposable,
self-hosted Temporal cluster. The test keeps one Go executor process and one cluster alive while it
runs the normal caller-closure contract and its duplicate-delivery negative control. This qualifies
the resident-executor architecture without requiring a Go test process, a Lean process, or a fresh
cluster for each verification.

## Overview

Add a protobuf portable-evaluation IR, a Lean-owned compiler into that IR, and a bounded Go
interpreter. Compose the interpreter with the existing runner behind one deep executor module. Add a
thin HTTP adapter and qualify it against a disposable `testcore.NewEnv` cluster using pre-generated
normal and negative-control contracts. The executor returns existing detailed stage statuses plus a
local canary decision of `pass`, `fail`, or `inconclusive`.

## Goal & Context
<!-- scope: business -->

A canary cannot depend on Lean being installed or invoke `lake`, Make, or a generated Go test for
each verification. It must receive a closed, inspectable contract, execute it, wait for bounded
Evidence closure, and decide locally. The decision is deliberately limited to the exact Test and
Evidence supplied by that contract. Model consistency, exhaustive search, cross-test coverage,
contract-generation correctness, release eligibility, and other whole-world claims remain offline
Lean/build-time responsibilities.

## Architecture & Data Models
<!-- scope: technical -->

```text
Lean model/build time
  -> closed per-test evaluation value
  -> canonical ProtoJSON
  -> structural Go packer
  -> deterministic EvaluationContract protobuf bytes

resident Go executor (no Lean)
  -> strict contract admission
  -> existing bounded runner and participant
  -> explicit Evidence closure or deadline
  -> portable Observation / Implementation Link / Property interpreter
  -> detailed Result + local pass/fail/inconclusive decision
```

The protobuf schema lives at
`proto/internal/temporal/server/api/umpire/v1/message.proto` and generates the conventional
`api/umpire/v1` Go package. The schema is an inert transport: semantic clauses originate only from
Lean and carry Definition IDs, Behavior Fingerprints, source bindings, Limits, Known Gaps, and a
checksum over deterministic protobuf bytes. Unknown versions, fields, operators, enum values, or
binding drift fail before runtime I/O.

The IR is closed over one Test. It contains only the finite vocabulary needed to:

- bind the exact Experiment and RuntimeConfiguration;
- normalize admitted Evidence fields and dispositions;
- require source identity, correlation, causal/source-local order, cardinality, and closure;
- construct the selected Evidence-backed System trace;
- apply its exact Implementation Link;
- evaluate its exact Property clauses; and
- retain independent operational, Observation Evaluation, Implementation Link, Property, cleanup,
  and tooling statuses.

The operator vocabulary is data-only and deliberately small. It admits no arbitrary Go, Lean,
shell, callback, regex engine, network target, credential, or runtime extension. A contract whose
model semantics cannot be lowered into the supported vocabulary is non-portable and cannot produce
a canary success.

### Version-one operator table

Version one contains only operators exercised by the normal caller-closure and duplicate-delivery
contracts. Lean rejects every other checked Observation, link, or Property construct as
non-portable.

| Layer | Operator | Input and result | Exact semantics |
| --- | --- | --- | --- |
| Observation | `literal_text`, `literal_natural` | none → tagged text/natural | Preserve the literal exactly; naturals are unsigned values bounded by the contract's numeric Limit. |
| Observation | `field` | one declared kind/field → its tagged value | Exactly one matching field returns its value; missing is `unknown`, duplicate is `conflict`, and undeclared kind/type is `unsupported`. |
| Observation | `natural_render_v1` | natural → text | Render unsigned base-10 ASCII with no leading zero except `0`; another operand type is `unsupported`. |
| Observation | `present` | one expression → boolean | `false` only when the referenced field/binding is absent; malformed, duplicate, or mistyped input retains its `conflict`/`unsupported` diagnostic rather than becoming false. |
| Observation | `equals` | two values of the same tagged type → boolean | Exact typed equality; there is no string/natural/boolean coercion. A type mismatch is `unsupported`. |
| Observation | `all`, `any` | nonempty ordered boolean operands → boolean | Evaluate left-to-right without semantic short-circuiting so every referenced input is validated; return conjunction/disjunction only after all operands succeed. |
| Trace | `emit` | one condition and one value expression → one bound coordinate | For each source record of the declared kind whose condition is true, emit the declared Definition ID/kind/value at the contract-fixed coordinate. Missing required coordinates are `unknown`; duplicate or contradictory coordinates are `conflict`; extra coordinates are `conflict`. |
| Link | `rename_exact` | one exact source Definition ID/kind/value → one exact destination tuple | Apply only the bundled finite mapping. Missing source mapping is `unknown`; duplicate/contradictory mapping is `conflict`; undeclared vocabulary is `unsupported`. |
| Property | `per_step_implies` | trigger pattern, required pattern → boolean clause | For every admitted step, either the trigger does not match or at least one required value in that same step matches. The normal/negative contracts retain their authored `transitionContract` or `inputOutput` clause kind as provenance, but both lower to this operator. |
| Pattern | `equals_text` | text → boolean | Match the exact Definition ID, trace field, and text value. No coercion. |
| Pattern | `natural_at_most` | canonical natural and bound → boolean | Match the exact Definition ID/field, then compare unsigned naturals. Noncanonical or out-of-range values are `unsupported`, not a Property violation. |

Every expression visit, rule/record candidate, emitted coordinate, link entry, clause/step pair, and
pattern/value candidate costs one evaluation work unit in addition to the separately bounded input
records and bytes. Work is charged before evaluation; exact N is admitted and N+1 returns the
responsible Limit status. Expression depth and collection cardinalities also have explicit Limits.
The interpreter never relies on map iteration order: protobuf order is canonical and diagnostics use
the first failing item in that order. Task `.5` proves Lean/Go parity for every row, success/failure
branch, type mismatch, empty/missing input, and exact N/N+1 boundary, not merely the two final canned
verdicts.

### Deep executor module

The external seam is one transport-independent interface conceptually equivalent to:

```go
Execute(context.Context, Request) (Result, error)
```

`Request` supplies admitted contract bytes and one exact admitted input set. The executor fills a
fresh opaque run identity only after single-flight admission succeeds. `Result` contains the complete
execution/evaluation record and local decision. Contract admission,
runner invocation, Evidence closure, portable evaluation, cleanup checking, and decision mapping
stay inside the module. HTTP is a thin adapter over this seam. The disposable-cluster and existing
loopback authorities are adapters below the same executor; callers do not orchestrate its phases.

The resident process may handle multiple bounded requests without restarting. Fn-28 does not add a
fleet scheduler, lease service, crash recovery, persistent queue, production deployment, or release
controller. Those can later reuse the executor interface without changing contract semantics.

## API Contracts
<!-- scope: technical -->

- Lean lowers a checked, selected Test into a complete per-test value. A Go packer may validate and
  deterministically encode that value, but it performs no semantic selection or lowering.
- Canonical contract bytes use deterministic protobuf serialization. Admission rejects unknown
  fields and enum values, unsupported major versions/operators, noncanonical bytes, invalid Limits,
  checksum mismatch, duplicate identities, and crossed Definition-ID/fingerprint bindings.
- The Go evaluator interprets only the bundled Observation mapping, trace projection,
  Implementation Link, and Property clauses. It never consults a model registry or synthesizes a
  missing clause.
- Evidence absence may establish a fact only after every contract-required source has explicitly
  closed. Wall-clock quiet periods never establish absence.
- Eventual Evidence collection uses contract-bounded source closure, cursor/watermark or terminal
  receipts, and a deadline. Reaching the deadline without closure returns an incomplete operational
  stage and `inconclusive`; it never returns `pass`.
- The detailed semantic statuses remain `satisfied`, `violated`, `unknown`, `conflict`, or
  `unsupported`. Canary mapping returns `pass` only for a fully closed successful run with accepted
  Observation Evaluation, applied Implementation Link, satisfied Property, and complete cleanup;
  returns `fail` only for a trustworthy closed `violated` verdict; and returns `inconclusive` for
  all operational, closure, unknown, conflict, unsupported, or tooling failures.
- The HTTP adapter accepts bounded protobuf request bytes and returns bounded protobuf result bytes.
  It exposes no arbitrary executable, checker path, Lean command, model selector, endpoint,
  credential, retry, or semantic override.
- One executor is single-flight. Its atomic state is `idle`, `active`, or `poisoned`. An overlapping
  request observes `active` and returns a typed `busy` tooling result plus local `inconclusive`
  before runtime I/O. State returns to `idle` only after complete cleanup; uncertain cleanup moves
  permanently to `poisoned`, whose requests also fail before runtime I/O. HTTP concurrency cannot
  bypass this admission.
- `testcore.NewEnv` owns the disposable cluster and SDK client. The Umpire adapter owns and cleans up
  only per-run workers, endpoints, workflows, correlations, and other run-created resources; it
  never closes or mutates the enclosing test harness outside the run contract.

## Edge Cases & Constraints
<!-- scope: technical -->

- Dynamic run IDs, workflow IDs, task queues, timestamps, and transport record IDs are normalized
  through contract-declared slots and correlations; expected output is stable semantic data, not a
  canned byte-for-byte runtime trace.
- Canned normal and negative-control contract protobufs are generated before the tagged test. The
  test process has no Lean dependency and does not shell out to Make, `mise`, `lake`, or `go test`.
- Duplicate, missing, ambiguous, causally unrelated, stale, unsupported, or post-closure Evidence
  fails closed according to the existing independent status model.
- One resident executor and one `testcore.NewEnv` cluster run both live contracts sequentially with
  fresh run isolation, proving process/cluster reuse. A poisoned or uncertain cleanup state prevents
  another request from being accepted by that executor instance.
- Overlapping HTTP requests are never queued or run concurrently. The loser receives the typed
  pre-I/O `busy`/`inconclusive` result; cancellation and cleanup of the active request complete
  before another request can observe `idle`.
- Cancellation closes every owned resource and returns an honest incomplete result. The executor
  never silently redispatches a possibly-started Test.
- Work and memory are limited independently for contract bytes, operators, Evidence records,
  Evidence collection, evaluation, HTTP bodies, and total request duration. N is admitted and N+1
  fails at the responsible seam.
- Existing comments in changed source remain intact.

## Quick commands

```bash
make proto
cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests
go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/...
go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$'
make lint-model
make umpire-check-regression
make lint-code
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A versioned protobuf `EvaluationContract`, the exact version-one operator table above, and
  result vocabulary live under the conventional
  internal Umpire proto package, generate normal Go bindings, serialize deterministically, and fail
  closed on unknown or noncanonical input.
- **R2:** Lean is the only semantic compiler. It lowers the exact selected Test, Observation,
  Implementation Link, Property clauses, Limits, Known Gaps, Definition IDs, and Behavior
  Fingerprints into a closed per-test contract; Go performs only structural packing and execution.
- **R3:** A generic Go interpreter with no Lean dependency reconstructs the Evidence-backed trace,
  applies the bundled link, evaluates the bundled clauses, and produces the existing independent
  detailed statuses plus a correct local canary decision.
- **R4:** Evidence collection supports eventual consistency only through explicit bounded closure.
  Complete positive and negative Evidence decides; deadline, ambiguity, conflict, unsupported data,
  missing closure, or uncertain cleanup is inconclusive and never success.
- **R5:** One deep resident executor module hides atomic single-flight admission, execution, Evidence
  closure, evaluation, and cleanup behind one small interface; a thin bounded HTTP adapter can serve
  multiple sequential requests without starting a new Go or Lean process and rejects overlap before
  runtime I/O.
- **R6:** One tagged Go integration test using `testcore.NewEnv` keeps one disposable cluster and one
  executor process alive, runs the pre-generated normal and duplicate-delivery contracts with fresh
  isolation, and observes respectively a local pass and a trustworthy local fail without invoking
  Lean.
- **R7:** The contract and result explicitly limit the local verdict to one exact Test. Whole-model
  validity, exhaustive coverage, compiler correctness, release eligibility, and cross-test claims
  are neither evaluated nor implied by the canary.
- **R8:** Mutation tests reject crossed bindings, checksum/fingerprint drift, unknown operators,
  Limit plus one, malformed closure, post-closure Evidence, stale run correlations, oversized HTTP
  messages, overlapping admission, cancellation leaks, and reuse after uncertain cleanup without
  partial success.

## Early proof point

Before adding HTTP or the `testcore` adapter, generate the normal caller-closure protobuf contract
from Lean and prove that the Go interpreter returns the same detailed Run Evaluation statuses and
stable semantic outcome as the existing Lean checker for both normal and duplicate-delivery canned
Evidence. If parity cannot be established with the closed IR, the spec stops there rather than
adding test-harness or resident-process machinery around an untrusted evaluator.

## Boundaries
<!-- scope: business -->

- No production canary deployment, fleet scheduler, lease system, persistent queue, crash recovery,
  checkpoint/resume controller, release decision, Claim Assessment, or automatic promotion.
- No arbitrary code, dynamic checker, model registry, Lean installation, or model source in the
  resident Go executor.
- No whole-world verdict. The executor decides only the exact admitted Test under its bundled
  Evidence policy, Limits, Known Gaps, and bindings.
- No replacement of specialized unit, race, persistence, schema, authorization, or performance
  tests.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

Precompiling the semantic checks makes canary execution autonomous while preserving Lean as the
only behavioral authority. A resident executor amortizes process and cluster startup and supports
Evidence that arrives after the workload has completed.

### Implementation Tradeoffs
<!-- scope: technical -->

A declarative protobuf interpreter adds a small trusted Go kernel and a versioned IR, but avoids
per-Test compilation, code injection, and deployment churn. Generated Go assertion code was rejected
because every contract change would require rebuilding the canary. Shipping Lean or WebAssembly was
rejected because canary has no Lean runtime and would inherit a much larger runtime/toolchain trust
surface. Hard-coded checker profiles were rejected because the current caller-closure specialization
does not scale beyond one Test.

Using canonical ProtoJSON as the build-time handoff lets Lean own semantic lowering while generated
Go protobuf code owns only wire validation and deterministic binary encoding. The canonical runtime
artifact remains protobuf bytes; the intermediate JSON is not a second semantic source.

## References

- `.plans/UMPIRE4_SPEC.md` — authoritative model, Artifact, Evidence, and Run Evaluation rules.
- `.plans/UMPIRE4_ORDER.md` — prototype sequence and portability gate.
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — existing Artifact admission.
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — bounded runner lifecycle.
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — canonical Lean Run Evaluation.
- `.flow/specs/fn-27-hermetic-ci-execution-and-qualification.md` — generated Go and CI portability.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Protobuf IR, deterministic encoding, strict admission | `.1`, `.2` | — |
| R2 | Lean-owned per-test lowering | `.3`, `.5` | — |
| R3 | Portable Go evaluation and detailed verdicts | `.4`, `.5` | — |
| R4 | Explicit eventual Evidence closure | `.4`, `.6`, `.10` | — |
| R5 | Deep resident executor and HTTP adapter | `.6`, `.8` | — |
| R6 | Tagged `testcore.NewEnv` end-to-end proof | `.7`, `.8`, `.9` | — |
| R7 | Per-test-only claim boundary | `.3`, `.9`, `.11` | — |
| R8 | Fail-closed mutation and lifecycle matrix | `.2`, `.4`, `.10` | — |

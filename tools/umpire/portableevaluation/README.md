# Portable Evaluation

Portable Evaluation lets one precompiled Umpire Test execute and reach a local decision in a Go
process that has no Lean runtime. The fn-28 `EvaluationContract` and HTTP interface remain the
historical/current compatibility path: Lean is their only semantic compiler. The successor
`PortableTestPlan` and generated unary gRPC interface are caller-neutral. Lean is the first model
compiler into that closed typed plan, while any conforming protobuf client may author an external
plan whose authority is explicitly limited to plan-local conformance. In both paths Go admits,
runs, closes, and independently evaluates only the supplied contract.

The legacy protobuf schema is
[`proto/internal/temporal/server/api/umpire/v1/message.proto`](../../../proto/internal/temporal/server/api/umpire/v1/message.proto).
It generates the conventional `go.temporal.io/server/api/umpire/v1` package. The schema is inert
transport data; neither generated Go nor the interpreter selects model meaning.

## Build-time contract workflow

The complete handoff is:

```text
checked Lean Test + Observation + Implementation Link + Properties
  -> Temporal.Tool.PortableEvaluationContract
  -> canonical ProtoJSON
  -> evaluationcontract.Pack
  -> strict structural validation and checksum sealing
  -> deterministic EvaluationContract protobuf bytes
```

`model/Umpire/Artifact/PortableEvaluationContract.lean` owns the reusable closed data vocabulary
and canonical ProtoJSON spelling. `model/Temporal/Tool/PortableEvaluationContract.lean` is the
Temporal-owned semantic compiler for the normal caller-closure Test and its duplicate-delivery
negative control. Unsupported checked constructs return a source-bound `NonPortableError`; Go does
not approximate or replace them.

`evaluationcontract.Pack` accepts only exact canonical ProtoJSON, rejects unknown fields and enum
values, fills an absent checksum structurally, and emits deterministic protobuf bytes.
`evaluationcontract.Admit` rechecks the version, checksum, complete bindings, canonical order,
operator vocabulary, Limits, and deterministic encoding before runtime I/O. The contract binds one
exact `umpire-experiment/v2` and `umpire-runtime-configuration/v2`, not an environment selector.

## Version-one contract

`EvaluationContract` carries version `1.0`, a contract and checksum identity, exact Experiment and
RuntimeConfiguration bindings, one Test and Query binding, independent evaluation Limits, one
Observation program and Evidence profile, one exact-renaming Implementation Link, one or more
Properties, Known Gaps, and source provenance. The corresponding `EvaluationResult` retains the
contract checksum, run identity, work charges, Known Gaps, diagnostics, and these independent
statuses:

- tooling: `succeeded`, `invalid_contract`, `invalid_input`, `busy`, `poisoned`, `canceled`, or
  `internal_error`;
- operational: `succeeded`, `incomplete`, or `failed`;
- Observation Evaluation: `accepted`, `unknown`, `conflict`, or `unsupported`;
- Implementation Link: `not_evaluated`, `applied`, `invalid`, `unknown`, `conflict`, or
  `unsupported`;
- each Property and clause: `satisfied`, `violated`, `unknown`, `conflict`, or `unsupported`;
- aggregate evaluation: `satisfied`, `violated`, or `incomplete`; and
- cleanup: `complete`, `incomplete`, or `failed`.

One status never implies another. In particular, operational success is not Property satisfaction,
an Observation or Link failure is not a Property violation, and cleanup uncertainty is not success.

The version-one operator vocabulary is closed:

| Layer | Operator | Exact behavior |
| --- | --- | --- |
| Observation | `literal_text`, `literal_natural` | Produce an exact tagged literal; naturals are canonical unsigned base-10 values within the contract Limit. |
| Observation | `field` | Read one declared typed field; missing is `unknown`, duplicate is `conflict`, and undeclared or mistyped input is `unsupported`. |
| Observation | `natural_render_v1` | Render a natural as unsigned base-10 text, with no leading zero except `0`; another type is `unsupported`. |
| Observation | `present` | Return false only for an absent referenced field; malformed, duplicate, and mistyped input retains its diagnostic. |
| Observation | `equals` | Compare values of the same tagged type exactly, without coercion. |
| Observation | `all`, `any` | Evaluate a nonempty ordered Boolean list left to right without semantic short-circuiting. |
| Trace | `emit` | Emit one contract-fixed coordinate for each matching record; missing, duplicate, contradictory, or extra coordinates fail closed. |
| Link | `rename_exact` | Apply only the bundled finite source-to-destination value and definition mappings. |
| Property | `per_step_implies` | For every admitted step, require the bundled pattern when the trigger pattern matches. |
| Pattern | `equals_text` | Match one exact Definition ID, trace field, and text value without coercion. |
| Pattern | `natural_at_most` | Match one exact Definition ID and field, then compare a canonical bounded natural. |

There is no callback, arbitrary Go or Lean code, shell command, regex engine, registry lookup,
network target, credential, model selector, or extension hook in the contract. Adding an operator
requires a versioned schema/compiler/interpreter change; unknown operators fail admission or
evaluation and cannot produce `pass`.

## Limits and Evidence closure

The contract independently bounds contract and input bytes, Evidence records, expression depth,
total operators, collection size, natural values, evaluation work, diagnostics, Result bytes, and
total duration.
Every expression visit, rule/record candidate, emitted coordinate, Link entry, clause/step pair,
and pattern/value candidate is charged before evaluation. Exact Limit N is admitted and N+1 fails
at the responsible seam.

The current compiler emits ceilings of 1 MiB for the contract, 16 MiB for evaluation input, depth
64, 10,000 total operators, 10,000 collection items, natural `4294967295`, 64 KiB of diagnostics,
and 4 MiB for the Result.
Its Evidence-record Limit comes from the checked Observation plan, its work Limit is derived from
the selected Test's search Limit with a minimum of 1,000, and its total duration is the sum of the
RuntimeConfiguration phase Limits. Go also enforces repository-wide hard maxima, so a contract
cannot broaden those ceilings.

Absence is usable only after explicit bounded closure. The contract lists every required Evidence
source. The runner returns an `ExperimentRun` source-closure record and matching Raw Evidence source
record for each source, including status, record count, and byte count. The executor waits for that
run closure, passes the expected closure to the evaluator, and rejects stale or mismatched run,
binding, source, count, or byte identities. The evaluator requires every declared source to be
closed before it constructs a trace and retains closure support in every accepted Evidence Link.
It never treats a wall-clock quiet period as closure. A deadline before closure, a partial or
missing closure, or Evidence added after closure is `inconclusive`, never `pass` or an invented
violation.

## Legacy resident executor and HTTP adapter

The transport-independent seam is:

```go
Execute(context.Context, *umpirev1.ExecuteRequest) (*umpirev1.ExecuteResponse, error)
```

One `executor.Executor` instance is single-flight. It atomically moves from `idle` to `active`,
admits the deterministic contract and exact two-member executable input, assigns a fresh opaque run
identity, invokes the existing bounded runner, waits for source closure, evaluates the result, and
checks cleanup. An overlapping request returns typed `busy` plus `inconclusive` before runtime I/O.
Complete cleanup returns the instance to `idle`; uncertain cleanup permanently moves it to
`poisoned`, and later requests fail before runtime I/O. Requests are never queued, silently retried,
or redispatched after a possibly started Test.

`tools/umpire/executorhttp` is a thin HTTP adapter over that seam. It is deliberately not gRPC:

- the only route is `POST /umpire/v1/execute`;
- request and response media type is exactly `application/x-protobuf`;
- bodies are deterministic `ExecuteRequest` and `ExecuteResponse` protobuf bytes;
- unknown fields, unknown enum values, malformed bytes, and noncanonical request bytes are rejected;
- the transport caps a request at one 1 MiB contract plus two 32 MiB artifact documents and a
  small protobuf envelope, caps the Result at 4 MiB, and enforces a five-minute outer deadline;
  the admitted contract may tighten its own bounds; and
- transport failures use HTTP status without fabricating an evaluation Result, while admitted
  executor failures use the detailed typed Result.

The adapter exposes no executable path, model/profile selector, environment endpoint, credential,
retry policy, semantic override, or deployment control.

## Caller-neutral plan and gRPC executor

The successor schema is
[`proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto`](../../../proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto).
It carries the complete bounded execution and verification programs in one `PortableTestPlan`.
External plans need no Lean or host provenance verifier and produce only `plan_local` results.
Lean-generated `model_compiled` plans use the same message and produce `model_bound` results only
after the executor host matches the exact checksum and model/compiler provenance. Missing, forged,
or crossed model provenance fails before runtime I/O and is never downgraded.

`tools/umpire/executorgrpc` implements only the generated unary
`UmpireExecutor.Execute(PortableTestPlan) -> ExecutionResult` method over one resident
`executor.PortableExecutor`. It preserves typed results after admission. Failures that prevent a
result use canonical gRPC status: malformed/crossed input is `INVALID_ARGUMENT`, unsupported
behavior or provenance and poisoned reuse are `FAILED_PRECONDITION`, hard bounds and overlap are
`RESOURCE_EXHAUSTED`, caller cancellation and deadline retain their corresponding statuses, and
server invariants are `INTERNAL`. The server queues and retries nothing. One admitted call runs at
a time; cleanup completes after client cancellation, uncertain cleanup poisons the resident
executor, and each reusable call receives fresh run resources.

The tagged `TestUmpirePortableGRPCExecutor` proof uses one disposable `testcore.NewEnv` cluster, one
resident in-process gRPC server, and a real generated client. It consumes the checked-in
Lean-generated plans plus a derived external-author variant after the test removes all toolchain
executables from `PATH`. The Go executor independently runs and evaluates normal pass and
trustworthy negative-fail outcomes, closure/crossed controls, ten-call overlap, cancellation and
deadline cleanup, poison, malformed/forged provenance, exact bounds, and fresh-run isolation. It
does not launch Lean, a shell, or a nested `go test` for any verification.

Fn-29 owns the production handoff. Its protected controller pins and provenance-validates one
Lean-generated plan before calling this Umpire gRPC interface. The separate public Temporal gRPC
connection remains the runtime adapter's downstream target; production credentials, target
selection, fencing, recovery, publication, and retry policy do not enter this reusable executor.

## Local decision

The evaluator maps the independent statuses conservatively:

- `pass` requires tooling and operation success, accepted Observation Evaluation, an applied
  Implementation Link, aggregate semantic `satisfied`, and complete cleanup;
- `fail` requires the same trustworthy closed execution and aggregate semantic `violated`; and
- every other combination is `inconclusive`, including invalid input, operational or tooling
  failure, missing closure, unknown, conflict, unsupported data, cancellation, Limit exhaustion,
  incomplete evaluation, and cleanup uncertainty.

This decision covers only the exact Test, artifacts, Evidence policy, Limits, Known Gaps, and
bindings carried by the admitted contract. It does not establish whole-model validity, exhaustive
coverage, compiler correctness, cross-Test consistency, release eligibility, or a Claim Assessment.

## Stable and runtime-scoped fields

Parity compares stable typed semantic meaning: contract and artifact bindings, Definition IDs and
Behavior Fingerprints, mapping/Link/Property identities, Model Trace values and coordinates,
applied dispositions, source-local and causal ordering support, detailed stage/Property/clause
statuses, Limits, Known Gaps, cleanup, and the local decision.

A live execution intentionally fills fresh runtime-scoped data. Executor run IDs, workflow IDs,
task queues, per-run Nexus endpoint names, operation correlations, Evidence record IDs, and
timestamps may differ between executions. The tagged proof requires fresh correlations and
resources while retaining the same namespace, cluster, contracts, and stable semantic result; it
does not compare live runs byte for byte.

## Fixtures and tests

The normal, duplicate-delivery, and complete-operator fixtures under `testdata/` are generated
before runtime tests. Fixture generation invokes Lean to compile canonical ProtoJSON, uses the Go
packer to produce `contract.pb`, and records Raw Evidence plus Lean Run Evaluation oracles. The
checked-in protobuf bytes are the runtime artifacts; ProtoJSON is only the build-time handoff.
The `portable-test-plan-v1` subtree additionally contains the sealed normal, duplicate-delivery,
and required-obligation plans consumed directly by the caller-neutral Go executor.

Generate fixtures deliberately:

```sh
make umpire-gen-portable-evaluation-fixtures
```

Check for compiler, packer, oracle, or checked-in fixture drift without modifying the repository:

```sh
make umpire-check-portable-evaluation-fixtures
```

Focused verification is:

```sh
cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests
go test -count=1 -tags test_dep \
  ./tools/umpire/evaluationcontract/... \
  ./tools/umpire/portableevaluation/... \
  ./tools/umpire/executor/... \
  ./tools/umpire/executorhttp/...
go test -count=1 -tags 'test_dep integration' ./tests \
  -run '^TestUmpirePortableCanaryExecutor$'
go test -count=1 -tags 'test_dep integration' ./tests \
  -run '^TestUmpirePortableGRPCExecutor$'
```

The tagged test creates one disposable `testcore.NewEnv` cluster, borrows its SDK client and
namespace through `local.NewAttachedFactory`, and keeps one resident executor and HTTP server alive.
It runs the pre-generated normal and duplicate-delivery contracts sequentially with fresh run
isolation, observing `pass` and the trustworthy uniqueness-only `fail`. It also proves crossed
input rejection and cleanup of every per-run worker and Nexus endpoint. After test compilation it
replaces `PATH` with an empty directory, proving runtime evaluation does not invoke Lean, `lake`,
`mise`, Make, a shell, or a nested Go test. `testcore.NewEnv` retains ownership of the borrowed
cluster and client; Umpire owns only resources created for one run.

This proof is not fleet scheduling, a lease service, persistence, crash recovery, production
deployment, automatic promotion, release eligibility, or Claim Assessment. Those remain separate
work and must not be inferred from a local `pass` or `fail`.

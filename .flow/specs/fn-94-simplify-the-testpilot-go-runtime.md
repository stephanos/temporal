# fn-94-simplify-the-testpilot-go-runtime Simplify the Testpilot Go runtime

> HTML render lens (local): open `.flow/artifacts/fn-94-simplify-the-testpilot-go-runtime/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

This spec is a simplification campaign over the handwritten Testpilot Go code. It is the Go
counterpart of fn-93, which covers the Lean model. It adds no concept, changes no Run or Verdict
semantics and keeps the facade API: `Prepare`, `PreparedCase.Run`, `Driver` and `Session`, in the
shape fn-91 leaves them. Generated code is out of scope: `api/testpilot/v1` and the Case JSON
fixtures that `make umpire-gen-case-runtime-conformance` writes.

SEM-16, SEM-17, EVD-01, EVD-04, EVD-12 to EVD-21, ART-09 to ART-14 and MOD-* hold unchanged. The
campaign rests on SEM-16 and EVD-19: admission is the one authority over what a prepared Case may do
before Driver I/O. Code that re-validates admitted data is therefore duplication. The Driver's
per-call authority checks are not, and they stay.

One lane changes the wire: lane G, only for the arms the owner's D1 decision removes. Removing a oneof arm or field from the hand-written
`.proto` removes a Program or Contract capability. ART-04 needs no migration, because the wire has
no compatibility promise and no Case outside the repository exists (fn-87's decision, restated by
fn-89). SCP-01 argues for removal, since no concrete Temporal use case produces these arms today.

Every `.proto` edit moves the Driver catalog identity, because the Driver catalog folds the
`case.proto` and `run.proto` closures in. A wire task therefore runs `make
umpire-rerecord-pinned-runs` (live cluster) and commits the refreshed records, receipts and identity
pins in the same commit. Outside that wire task, the catalog identity, `BindingFingerprint`
bytes and route wire bytes do not move.

## Goal & Context
<!-- scope: business -->

The handwritten Testpilot Go code is 42,067 lines (measured 2026-09-26):

| Part | Lines |
| --- | --- |
| `common/testing/testpilot` production (`ir`, `execution`, `verification`, facade, `contract`, `temporal/*`) | 18,340 |
| `common/testing/testpilot` tests | 17,539 |
| `tests/testcore/testpilot` Go (468 production fixture code, 2,049 tests) | 2,517 |
| `tests/testpilot_*_test.go` live tests (13 files) | 2,289 |
| Hand-written `.proto` (`proto/internal/temporal/server/api/testpilot/v1`) | 1,382 |

fn-89 and fn-90 have since grown it. The R10 baseline is re-measured at HEAD `cfb29fc045`
(2026-09-27) with the Measurement commands below: production (core plus `tests/testcore/testpilot`)
19,332, tests 20,550, live tests 2,399, `.proto` 1,414. fn-94.2 re-measures it when it starts and
records that as the baseline every later receipt compares against.

Four read-only investigations found this code already fairly tight. Two runs of the `deadcode` tool
found only about 40 unreachable lines and about 45 test-only ones. The bulk that can go falls into
five kinds:

1. **Residue of the retired untyped Nexus path.** The Worker Driver still carries the `Value`-based
   Nexus result, its fallback future decoding, header and value parameters that the one production
   caller passes as `nil`, and three reserved-header merges where one would do.
2. **Re-validation of admitted data.** Examples:
   - Environment bindings are validated in both the facade and `execution`, and the Profile is
     deep-cloned three times on the way in.
   - The worker and server Drivers re-check Profile ceilings that `execution.checkLimits` already
     checks.
   - Delivery re-checks the carrier topology that `execution/carrier.go` compiled.
   - The server rebuilds node indexes and re-resolves instruction defaults that `InstructionPlan`
     already carries.
3. **Primitives copied across packages.** Examples:
   - `nilValue` ×3, `contextError` ×2, three context-aware mutexes, three effect-result clones;
   - `invalid`/`validID`/`isNil` ×2–3, four copies of the ceiling-reflection loop;
   - hand-written `slices`, `maps` and `cmp` equivalents.
4. **Test sprawl.**
   - 12 test copies of the descriptor-closure body (named `descriptorClosure`,
     `facadeDescriptorClosure`, `nexusDescriptorClosure`, or inline).
   - 15 copies of one `ProgramLimits` literal.
   - A ~220-line runtime fixture copied between `activation` and `worker`; about 100 lines differ.
   - 10 fake Sessions, 13 fake Drivers and 16 per-file Case builders.
   - Three scripted sessions in `tests/testcore/testpilot` (`workflowStartSession`,
     `artifactSession`, `nexusPairGapSession`) that share most of their code.
   - Live-test run and assert boilerplate repeated three times.
5. **Dormant protocol surface.** Oneof arms that no Lean producer emits and no fixture uses, but
   that the Go runtime still handles. `Reference.evidence_field_id` has no producer at all, and
   `Reference.model_value` has no Go handler and no context that admits it.

One finding is a defect, not bulk. `CorrelatedContract.initial_state_fields` and
`CorrelatedTransition.prior_fields` are emitted by Lean and present in 14 fixtures. Go admission
neither validates nor reads them: the STATE step condition reads `State` plus `StateFields` only.
The investigation settled which. No rule condition can read a prior or initial state (there is no
prior step field), but Lean's correlated monitor does read both fields: it decodes every state as a
`StateValue` (the atom plus its fields), starts from the initial `StateValue`, and takes a transition
only when its prior `StateValue`, fields included, equals the current one. Go's monitor tracks the
atom alone, validates none of the three field lists (`state_fields` included), and its output-row
equality and continuity ignore fields where Lean compares them. For Lean-produced Cases the fields
follow from the atom, so the corpus agrees today; a Case whose fields disagree is where Go and Lean
diverge. EVD-04 (fail closed) makes each gap an issue.

Done means the Testpilot Go code is smaller and there is less of it to read. Every behavior the
corpus, the conformance tests and the live identities pin stays the same.

## Architecture & Data Models
<!-- scope: technical -->

Lane E comes first, then identity pins, dead code (A), validate-once (C), shared primitives (B),
the internals (D1, D2), tests (F) and the protocol arms (G). Lanes A, B, C, D1, D2 and F are
mechanical and need no decision beyond this spec. Lanes E, G and H carry owner decisions, each
recorded in its task with a recommended default the conductor takes. Each lane is one or more tasks,
each with its own receipt, which records the measurement below before and after.

### Identity pins (before any refactor that could move them)

No golden pins `BindingFingerprint` bytes, the Driver catalog identity or the delivery route wire
bytes today; only the pinned Run records carry the first two. One task adds a golden test for each,
and records the R10 baseline and the passing live identity count, before lanes C, D2 and G land.
The wire task updates the catalog golden in the same commit as its re-record; no other task may.

### Lane E: settle the unread correlated fields

- **E1.** Decision: read (see Owner decisions). Go does what Lean's decoder and monitor already do,
  with no wire, Lean or fixture change:
  - **Admission** validates every `state_fields`, `prior_fields` and `initial_state_fields` entry
    like `validModelValue`, and output-row equality compares `state_fields` as Lean's `Result` does.
  - **The monitor** carries a state as atom plus fields: it starts from `initial_state` with
    `initial_state_fields`, matches a transition on `prior_state` with `prior_fields`, and moves to
    `state` with `state_fields`. Candidate counting for work accounting uses the same match, so
    Lean-produced Cases keep their work totals.
  - Focused tests pin each rejection and one Case whose fields disagree with its atoms, where Go now
    behaves as Lean's `StateValue` equality does. The corpus stays unchanged, because Lean produces
    consistent fields.

### Lane A: dead and test-only code

- **Unreachable:**
  - `worker.Driver.prepareDefinitionPlans`
  - `worker/routing.go`'s `validCoordinate`; `delivery` keeps its own
  - `PreparedProgram.PolicyIdentity`
  - `InstructionPlan.Assignments` and `AssignmentPlan`, `InstructionPlan.Input`
  - `worker/carrier.go`'s `Handles` and `Quarantine`
  - the `prepared_case.go` type assertion to an anonymous `Evaluate` interface, whose error branch
    cannot happen. The prepared contract is stored beside the `MonitorFactory`, so no fake factory
    has to grow an `Evaluate` method.
- **Test-only production code**, deleted and its tests re-pointed at the production path:
  - `worker/carrier.go` `ParentTerminal`; production uses `Session.parentTerminal`
  - `Session.preparedNexusDispatch`
  - `workerLease.release`
  - `Path.CheckFanout`
  - `Outage.Stopped`
  - `delivery.Activation.Handle`
  - `ProgramView.MaximumActivations` and the computation that feeds it
  - `assignment.environmentBindingID`
  - `PreparedContract.Snapshot` and `PreparedContract.ProgramView`
  - `InstructionPlan.ResponseReads` and `ResponseReadPlan`
  - `Options.SessionOptions`/`Driver.Open`
  - the unprepared `newCompositeSession`
- **Facade surface with no caller in the repository:** `InstructionPlan.Guard`, `OutcomeType` and
  `Dependencies`, and the facade `Expression` type with `Expression.Evaluate`. The READMEs that
  document them are updated in the same commit. Of the six aliases no non-test code outside the
  root package uses (`ReferenceKind`, `OutcomeSnapshot`, `ReservationTopology`, `ReservationRoute`,
  `SlotReference`, `InjectFault`), an alias goes only if no remaining facade signature names it, so
  no `contract` type leaks into the public API.
- **Untyped Nexus residue:**
  - `nexusResult.value`
  - the always-true `typed` map and its `future.Get(&Value)` fallback
  - the `*testpilotspb.Value` input case and `Value`-to-payload encoding
  - `PrepareNexus`'s `header`/`value` parameters, and `NexusDispatch.value`/`Value()`
  - two of the three reserved-header merges
  - A typed sync reply with no payload keeps its current result; a test pins it before the
    fallback goes.
- **Dead test helpers** that `deadcode -test` reports:
  - `runtimeText`, `runtimeStatusType`, `runtimeTextType` in the `activation` fixture, and
    `runtimeStatusType`, `runtimeTextType` in the worker fixture (its `runtimeText` is used)
  - `openCountingDriver.Open`
  - `workflowAdmissionForTest`
  - the live-test trio `runCase`, `runCaseWithBinding` and `runBoundCase`, which `runCaptured*`
    duplicates. `runCase` has no caller; the one `runCaseWithBinding` caller moves to
    `runCapturedCaseWithBinding` without renaming its live test.

### Lane B: shared primitives

- **`ir` exports** `Invalid` (with the 256-byte path truncation), `ValidID`, `IsNil` and
  `CheckCeilings(limits, ceiling, path, optional...)`. `execution`, `verification`, the facade and
  the worker Driver call them in place of their copies.
  - `CheckCeilings` takes the error path and the fields to skip as parameters, so `execution`'s
    field-name path, `verification`'s `"contract"` path with the field name in the detail, and the
    correlated copy's skipped fields all stay as they are.
- **One new helper package under `temporal/internal`** holds `nilValue`, `contextError`, the
  context-aware mutex, the effect-result clone, `nexusHeaderBytes`, the method-path builder, the
  StartWorkflow path constant and `hasWorkerEntrypoint`.
  - `contextError` takes the caller's sentinel error, so each package keeps returning its own.
  - The mutex keeps the no-context `Lock()` that the server needs for completion; the worker's
    variant, which rejects a nil context, keeps that behavior.
  - Where two copies differ in signature (`nexusHeaderBytes`' int widths, `hasWorkerEntrypoint`'s
    argument), the shared one takes the narrower input and callers convert.
- **Standard library replacements:** `slices.Contains`/`DeleteFunc` for `sessionIn`,
  `removeSessionFrom` and `slicesContains`; `slices.Sorted(maps.Keys(...))` for `setKeys` and
  `nexusSetKeys` where the order is observable; `cmp.Or` for `firstError` only if no argument has a
  side effect (it evaluates every argument). An empty result keeps its current nil-or-empty shape.

### Lane C: validate once

- **Environment bindings.** The facade's `BindingFingerprint` is the one validator of environment
  bindings, because Drivers call it at construction outside `Prepare` and the preparation-error
  tests pin its `profile.environment_bindings` path. `execution.bindPolicy` stops re-validating what
  it receives from `Prepare`, which is its only caller. The fingerprint's bytes stay identical
  (ART-14), pinned by the identity golden.
  - `Prepare` clones the Profile once instead of calling `Snapshot().Snapshot()` and re-cloning it
    in `bindPolicy` and `bindRolePolicy`. A test pins that mutating the caller's `ProfileSpec` after
    `Prepare` changes nothing in the prepared Case.
  - One sorted-unique-IDs helper serves both loops in `profile.go`.
  - `execution.Profile` stops hand-mirroring `ProfileSpec`.
- **Driver ceiling checks.** These run once, at Driver construction, on the `ProfileSpec`; they are
  not re-validation of admitted data and they stay. The worker's `validWorkerProfile` and the
  server's partial copy call the exported ceiling check instead of their own loops. The worker's
  `validateSymbolicRoles` keeps only the worker-role-ID check and `profileRoleHasMethods`.
- **Carrier checks.** The plan-shape re-checks in `delivery`'s `validateTopology`/`validateRoutes`
  and the worker carrier's reservation count and duplicate checks go: `execution/carrier.go`
  compiled the plan they re-check. The task names, for each removed check, the admission check that
  already rejects the same shape. What is not a re-check stays: `validateHandles` and the expected
  handle map it consumes, the runtime `MaxRoutes` limit, and the worker carrier's workflow-entrypoint
  lookup. The plan-shape tests go with the checks.
- **Server node index.** The server Driver indexes `InstructionPlan` values instead of rebuilding
  node and evidence maps from the raw snapshot. Its sessions read `TimeoutMilliseconds()` and
  `MaxAttempts()` instead of re-resolving `InstructionDefaults`.
- **What stays.** The per-call checks that coordinate, role and method match the prepared node
  (`server/session.go:81-92`, and the worker's equivalents) are the Driver's authority boundary.
  They stay.

### Lane D1: execution and verification internals

- **Opcode table.** One `opcodes` table maps each opcode to its oneof arm name, entrypoint context,
  protocol-code flag, and bind and dataflow-bind functions. It replaces the per-opcode switches in
  `dataflow.go` (`InstructionOpcode`, `opcodeContext`, `bindInstruction`, `bindNodeDataflow`,
  `bindOutcomes`) and `scheduler.acceptEffect`.
  - Admission checks fire in today's order, because the corpus pins their paths. Before the table
    lands, a focused test pins the first rejection for Programs that carry two defects at once, so
    the order is pinned beyond the corpus's one-defect entries.
- **Guarded input evaluation.** One `node.evaluateGuarded` replaces the three "guard, then input"
  evaluations (`request.go`, `scheduler.prepareInput`, `InstructionPlan.EvaluateInput`), with the
  same work accounting.
- **Scheduler and runtime helpers.**
  - One `takeCompletion` and one non-blocking drain replace the five copies of the completion-drain
    `select` in the scheduler.
  - `execute` and `executeCleanup` stay separate, because EVD-14's stop semantics differ.
  - One close helper replaces the two identical cancel-and-close blocks in `runtime.go`; the final
    close, which also checks the close context, stays as it is.
- **Correlated.** `validCorrelatedLiteral` and `correlatedLiteralKind` become one switch.
  Correlated stays outside `ir`, as its README says.
- **Expression binding.** `ir.BindExpression` becomes `bindConditionedExpression(nil, …)`.
- **Aliases and narrowed interfaces.**
  - `resolvedRole` becomes `contract.PreparedRole`.
  - The facade's `RuleViolation` becomes an alias of `verification.Violation` once the internal
    `Kind` field is renamed `CorrelatedKind`.
  - `execution.Run` takes the one method it calls; `driverAdapter`'s unused `Identity`/`Validate`
    go.
  - Forwarding functions (`EntrypointKindOf`, `InstructionOpcode`, `EnvironmentBindingIDs`) become
    `var` aliases.

### Lane D2: Driver internals

- **Server.** `Open` and `OpenSession` become one. `InvokeRPC` and `PollRPC` share one
  `authorizeUnary` prelude. The two claim-validity blocks in the handle invoke path become one
  helper that runs under the lock; both call sites stay, because the second re-checks the claim
  after the lock is released and re-taken around `Accepts`.
- **Workflow binding.** One comparable, JSON-tagged `delivery.WorkflowBinding` replaces the five
  representations of it and serves as the route key. Requests are dynamic messages built by `ir`,
  so one extraction function in `delivery` reads the binding from a StartWorkflow request by
  descriptor; the composite Driver's Marshal/Unmarshal round trip and the second extraction path
  go, and both callers use the one function. The carrier call order (`CreateCarrier` before
  `PrepareRPC`) stays. The JSON tags keep the route wire format byte-identical, pinned by the route
  golden. The Nexus start fields keep carrying their header beside the binding.
- **Admit paths.**
  - `AdmitWorkflow` and `AdmitNexus` share one consume-and-admit tail.
  - `Driver.admitWorkflow` and `admitNexus` share one fan-out loop.
  - The Session's admission caches stay: they carry replay after `Session.Close` and the once-only
    completion state, which the ledger's stop-first admission does not.

### Lane F: tests

- **Shared helpers.** One exported test helper for descriptor closures replaces the 12 test copies,
  and one `ProgramLimits` fixture constructor replaces the 15 literals. The helper package imports
  only protobuf and generated code, never the facade, so `ir` and `execution` tests can use it
  without an import cycle.
- **Runtime fixture.** One runtime fixture, parameterized by reply-kind enum, command types and
  capture step, replaces the near-copies in `activation_test.go` and `runtime_fixture_test.go`.
- **Shared fakes.** An internal test support package holds one scripted fake Session, one fake Driver
  and fake effect and reservation types. The in-package facade tests, which cannot import a package
  that imports the facade, keep one local copy each. Package-local fakes stay only where they test
  something specific:
  - `countingMonitorFactory`, the seam that `MonitorFactory` exists for;
  - the scheduler host;
  - the canary's fenced session.
- **`tests/testcore/testpilot`.** `workflowStartSession`, `artifactSession` and
  `nexusPairGapSession` become one scripted session with namespace, queue and history hooks. The
  unchanged-bytes checks share one helper.
- **Two-layer duplicates.** Where the facade and `execution` test the same environment rejections
  and concurrent preparations, the facade test stays. Exact duplicates go; focused tests that
  EVD-18 keeps independent of the corpus stay.
- **Live tests.** One helper covers the repeated-run assertion: distinct Run IDs, the
  workflow in its own namespace, Case bytes and frozen bindings unchanged. It serves the start,
  pair and caller live tests.
  - `nexusEvidenceKind` becomes a map, and `nexusOperationCoordinates` reads through a getter
    interface.
  - No live test is renamed, merged or removed, because fn-90 names these identities.

### Lane G (D1): dormant protocol surface

Remove the oneof arms that no Lean producer emits and no fixture uses, together with their Go
handlers, Lean `Testpilot.Authoring` builders and Lean `Testpilot.Correlated` cases. The per-arm
decision and its recommended default (see Owner decisions):

| Arm | Finding | Default |
| --- | --- | --- |
| `Reference.evidence_field_id` | No producer, no fixture; Go and Lean correlated handlers only | Remove |
| `Reference.correlated_capture` | No producer beyond its builder, no fixture; correlated handlers only | Remove |
| `Reference.model_value` | No Go handler; no context admits it | Remove |
| `RunDiagnostic.supporting_event_sequence` | The recorder sets it and the conformance test reads it | Keep (not dormant) |
| `SingularType.enumeration` | fn-89's instance values and captures use it | Keep (blocked, R9) |
| `ValueType.repeated`/`map`, `SingularType.any`, `Expression.any` | `ir` path fan-out, type checks and fn-89's instance expansion use them | Keep (not dormant in Go) |
| `Deadline.elapsed_milliseconds` | EVD-21 says it "remains admitted" | Keep (governed) |
| `InstructionLimits.max_attempts` | SEM-16 names instruction attempts; the server reads `MaxAttempts()` | Keep (governed) |
| `Entrypoint.activity` | MOD-13 and the Program glossary name activity entrypoints | Keep (governed) |

A kept arm needs a GOV-02 amendment to go, which is out of scope for a simplification campaign.
Before each removal, the task re-checks every open spec (fn-79's deferred cancellation scope,
fn-92's composition, fn-93) and every Lean producer for the arm.

Each removal regenerates the Go protocol code with `make proto` in the same commit; the Lean
protocol is elaborated from the `.proto` directly and is checked by `make
umpire-check-testpilot-protocol`, and the corpus by `make umpire-gen-case-runtime-conformance`. It
follows the extension checklist's numbering rule, adds a removal subsection to that checklist, adds
the compound names (for example `EvidenceFieldId`, `CorrelatedCapture` and their descriptor message
names) to the retired-vocabulary gate where SEM-20 and the gate's token grammar allow, and re-records
the pinned Runs. A name whose spelling a live identifier shares is not gated, and the task says so.
Case provenance stays: the Provenance restatement makes those rows part of every Case, and "the
runtime reads none of it" by design.

### Lane H (D2): facade wrapper structs

`driver.go`'s `PreparedProgram`, `EntrypointPlan` and `InstructionPlan` wrap `execution` types method
by method, in about 100 lines. The `contract` aliases are structural, because they break the
`execution` to facade import cycle, and stay. The recommendation is to keep the wrappers: they are
the public read-only view Drivers program against, and exporting `execution` types instead would
widen the facade. Lane A still removes their uncalled methods.

### Measurement

Every receipt uses these commands:

```sh
find common/testing/testpilot tests/testcore/testpilot -name '*.go' ! -name '*_test.go' | xargs cat | wc -l
find common/testing/testpilot tests/testcore/testpilot -name '*_test.go' | xargs cat | wc -l
cat tests/testpilot_*_test.go | wc -l
cat proto/internal/temporal/server/api/testpilot/v1/*.proto | wc -l
```

## API Contracts
<!-- scope: technical -->

The facade keeps `Prepare`, `PreparedCase.Run`, `PreparedCase.Evaluate`, `Driver`, `Session`, the
handle seam in its fn-91 shape, `ProfileSpec`, `BindingFingerprint` and the preparation error
categories. The shapes below are illustrative. The tasks settle exact names.

```go
// package ir (internal): shared admission primitives
func Invalid(path string, format string, args ...any) error // truncates path at 256 bytes
func ValidID(id string) bool
func IsNil(value any) bool
func CheckCeilings(limits, ceiling any, path string, optional ...string) error

// package testpilot (facade): removed, no caller in the repository
// InstructionPlan.Guard, InstructionPlan.OutcomeType, InstructionPlan.Dependencies,
// type Expression, Expression.Evaluate, and six unused aliases

// package delivery (internal): one binding type, route wire format unchanged
type WorkflowBinding struct { // the fields and JSON tags of today's codec.go binding
	Namespace    string `json:"namespace"`
	WorkflowID   string `json:"workflow_id"`
	WorkflowType string `json:"workflow_type"`
	TaskQueue    string `json:"task_queue"`
}
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **The corpus is the oracle.**
  - `make umpire-check-case-runtime-conformance` must show no diff in any `expected.json` or Case
    fixture.
  - The corpus pins each rejection's `category` and `path`. Error text may change, but the paths and
    the order in which admission checks fire may not.
  - The lane G regeneration is the only allowed fixture change, and only for the removed arms.
- **Identity bytes.** `BindingFingerprint`, Driver identity (ART-14) and the delivery route wire
  format stay byte-identical, pinned by goldens that land before any lane that could move them. The
  Driver catalog identity moves only in the lane G commit, which updates its golden and re-records
  the pinned Runs in the same commit.
- **Concurrency.** Changes to the scheduler, delivery and ledger run with `-race`
  (`go test -race -tags test_dep`) and keep the EVD-14 and EVD-15 tests green. `execute` and
  `executeCleanup` stay apart, and so do the recorder's `publish`/`publishCleanup` and
  `admit`/`admitCleanup`. The same goes for `ir.inspect`'s two modes and `newEvaluator`'s
  view-equality guard.
- **Authority split.** No lane moves a check from the Driver's per-call authority boundary into
  admission, or back. Server and worker authority stay separate (EVD-17).
- **Concurrent specs.**
  - fn-90 and fn-91 are delivered; nothing waits on them any more.
  - fn-89 changes Contract Rule evaluation and preparation in `verification`. Its code has landed;
    fn-89.5's receipt and fn-89.6's docs remain. Tasks that edit `verification` or change the wire
    wait for fn-89.5, and tasks that edit the root Testpilot README (the extension checklist) or
    `internal/verification/README.md` wait for fn-89.6.
  - fn-92 wants the caller, Control and canary fixtures byte-identical while it runs, and fn-93
    wants every fixture byte-identical and no Testpilot wire change while it runs. So the wire task
    (lane G) starts only when fn-92 and fn-93 each either have no task started or have closed; the
    task checks with `flowctl show` before it starts and stops otherwise.
  - Lane G edits `Testpilot/Authoring.lean`, the correlated Lean files and the retired-vocabulary
    gate, which fn-93 also edits; the same start rule covers that overlap.
  - fn-70's resume note relies on the test-local `runCase`; the final task updates that note in the
    order document when lane A removes it.
- **No new dependencies.** protovalidate is not in `go.mod`, and it would lose located paths and
  work accounting anyway. `deadcode` is used only as an offline investigation aid.
- **Explicitly not done.**
  - Table-driving the three small instruction-to-SDK switches, which would add code.
  - Typing role, slot or entrypoint IDs, which would ripple through `Coordinate` and the contract
    fn-91 edits.
  - Mapping context to outcome in one place: the three mappings use different codes.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** E1's decision is recorded in its task, and no field of an admitted correlated Case is left
  unvalidated or ignored where Lean reads it: every `state_fields`, `prior_fields` and
  `initial_state_fields` entry is validated like a model value, and Go's monitor compares states as
  atom plus fields wherever Lean's `StateValue` equality does (initial state, transition match,
  output-row equality, continuity). No wire, Lean or fixture change. Errors:
  - an entry in any of the three lists that is not a valid model value is rejected, pinned by a
    test for each list;
  - an output row whose `state_fields` differ from every authorized transition's is rejected,
    pinned by a test;
  - a Case whose `prior_fields` or `initial_state_fields` disagree with the fields the run reaches
    takes no transition that Lean would not take, pinned by a test;
  - the corpus shows no diff, including the work totals of every Lean-produced Case.
- **R2:** None of the declarations lane A lists remains. `deadcode -test -tags 'test_dep integration'`
  over `./common/testing/testpilot/... ./tests/... ./tools/...` reports no unreachable function in
  Testpilot packages. Errors: a declaration it still reports is deleted or given a production
  caller, never suppressed.
- **R3:** No Testpilot package keeps a private copy of a primitive `ir` or the `temporal/internal`
  helper exports, and no hand-written `slices`/`maps`/`cmp` equivalent remains. Errors: an error path
  or ordering change fails R8.
- **R4:** Admitted data is validated once. Environment bindings are validated only by
  `BindingFingerprint`, whose `profile.environment_bindings` paths stay pinned. `Prepare` clones the
  Profile once, and a test pins that mutating the caller's `ProfileSpec` afterwards changes nothing
  prepared. `delivery` and the worker carrier do not re-check the compiled plan's shape; each removed
  check names the admission check that already rejects that shape. The Drivers' construction-time
  ceiling checks call the exported ceiling check. The server reads node bounds and defaults from
  `InstructionPlan`. Errors: the Driver's per-call authority checks, `validateHandles` and the
  runtime `MaxRoutes` limit remain, each pinned by a rejection test.
- **R5:** One opcode table drives instruction binding and effect acceptance, and each other lane D1
  consolidation leaves one definition where there were several. Every static preparation rejection
  keeps its pinned `category` and `path`. Errors: the first rejection of a Program with two defects
  is pinned before the table lands and is unchanged after it.
- **R6:** One `delivery.WorkflowBinding` exists, route wire bytes are unchanged (the route golden
  passes unedited), one function extracts the binding from a request and every caller uses it, and
  no Marshal/Unmarshal round trip of the binding remains. Errors: a request that is not a
  StartWorkflow request, or lacks a binding field, keeps today's rejection, pinned by the existing
  delivery rejection tests.
- **R7:** Each shared test helper and fake has one definition, and every retained package-local fake
  says in one line what it tests that the shared one cannot. The live-test repeated-run helper serves
  the start, pair and caller tests, and the passing live identity count equals the count the
  identity-pin task recorded. Errors: no error surface beyond an import cycle, which the
  facade-free helper package rules out.
- **R8:** `make umpire-check-case-runtime-conformance`, `make umpire-check-testpilot-protocol` and
  `make umpire-check-testpilot-authoring` pass with no fixture or `expected.json` change outside
  lane G. `go test -race -tags test_dep ./common/testing/testpilot/...` passes,
  and so do `make umpire-check-regression` (the recorded count of passing live identities) and
  `make lint-code-fast`. Each task runs the focused gates its Quick commands name; the final task
  runs all of them. Errors: a gate that needs a live cluster or a cold Lean build is reported as
  not run with the reason, never as passed.
- **R9:** For D1, the per-arm decision is recorded before removal, with the defaults in the lane G
  table. Each removed arm is gone from the `.proto`, the regenerated Go, the Lean builders, decoders
  and renamer, and the Go handlers; its compound names are in the retired-vocabulary gate where
  SEM-20 and the gate grammar allow; the protocol extension checklist has a removal subsection; and
  the pinned Runs are re-recorded. Errors: an arm that turns out to be emitted by a producer, used by
  Go outside its handler, or named by an open spec or a governed requirement blocks its removal, and
  the task records it as kept.
- **R10:** The final receipt reports the measurement against the baseline fn-94.2 recorded (about
  19,332 production, 20,550 tests, 2,399 live, 1,414 `.proto` at `cfb29fc045`):
  - handwritten Testpilot production at least 6% smaller (~1,160 lines);
  - its tests at least 5% smaller (~1,030 lines);
  - the hand-written `.proto` smaller by the removed arms, reported without a floor,
    because the default decisions keep most arms.

  Errors: a floor missed is reported with the reason.
- **R11:** Golden tests pin `BindingFingerprint` bytes, the Driver catalog identity and the delivery
  route wire bytes before any lane that could move them lands. Only the lane G commit changes the
  catalog golden, and it re-records the pinned Runs with `make
  umpire-rerecord-pinned-runs` in the same commit. Errors: a golden diff in any other task fails
  that task; a re-record that cannot reach a live cluster blocks the wire task rather than landing
  stale records.

## Early proof point

fn-94.3 (the Go correlated monitor reading states as atom plus fields) proves the campaign's premise
on its one defect: Go can be brought to Lean's semantics with the corpus, work totals and identity
goldens unchanged. If a committed fixture diverges, the Lean producer and Go disagree somewhere the
corpus did not show; stop and re-evaluate E1 (removal would then need a Lean `StateValue` redesign,
coordinated with fn-93) before lanes that build on `verification`.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Correlated fields validated and read as Lean reads them, E1 recorded | fn-94.1, fn-94.3 | — |
| R2 | Lane A declarations gone; `deadcode` clean | fn-94.4, fn-94.5, fn-94.17 | — |
| R3 | One copy of each shared primitive; stdlib equivalents | fn-94.7, fn-94.8 | — |
| R4 | Validate once; authority checks stay | fn-94.6, fn-94.9, fn-94.10 | — |
| R5 | Opcode table and D1 consolidations; rejection order pinned | fn-94.11, fn-94.12 | — |
| R6 | One `WorkflowBinding`; route bytes unchanged | fn-94.13 | — |
| R7 | One shared helper and fake each; live count unchanged | fn-94.14, fn-94.15 | — |
| R8 | Gates green, fixtures unchanged outside lane G | every task (focused), fn-94.17 (all) | — |
| R9 | Per-arm decision; removed arms gone everywhere | fn-94.16 | — |
| R10 | Measurement against the recorded baseline | fn-94.2 (baseline), fn-94.17 (final) | — |
| R11 | Identity goldens; catalog moves only with a re-record | fn-94.2, fn-94.16 | — |

## Boundaries
<!-- scope: business -->

- No change to generated code (`api/testpilot/v1`, the generated Lean protocol) other than the lane G
  regeneration, and no change to the Case JSON fixtures other than its.
- No Run, Verdict, Monitor or Contract semantics change, and no new Opcode or Profile field.
- No facade API change beyond removing the uncalled methods and aliases lane A lists.
- `tools/umpire` and `tools/canary` are out of scope, except where they call a removed declaration.
  A sweep of them is a separate spec.
- No live test renamed, merged or removed.
- No Lean change outside lane G.

## Decision Context
<!-- scope: both -->

### How the campaign was scoped

Four read-only investigations covered the runtime core (facade, `contract`, `ir`, `execution`,
`verification`), the Temporal Drivers with `tests/testcore/testpilot` and the live tests, and a
mechanical dead-code and dependency scan. The scan used `deadcode` v0.49.0 offline from the module
cache, with grep cross-checks for methods and types that `deadcode` treats as live. The key claims
were re-checked by grep before writing:

- `prepareDefinitionPlans` and `PolicyIdentity` have no reference beyond their declaration.
- The worker's `validCoordinate` has no caller; `delivery`'s is a different function.
- `runCase` has no caller.
- `evidence_field_id` appears in no fixture.
- The only Go reader of correlated step fields is `predicate`, which reads `StateFields` only.

A planning pass on 2026-09-27 re-verified every lane at HEAD `cfb29fc045`, after fn-89's Go and
proto work, fn-90 and fn-91 had landed, and corrected the counts and claims above.

The Go code is tighter than the Lean model: this campaign is worth about 2,100 lines, against fn-93's
roughly 25,000. It is still worth running. Most of the value is in less code to read at the seam
every Driver implements, and in settling the unread wire fields.

### Owner decisions

Each decision is recorded in the task that owns it, with the recommended default the conductor
takes unless the owner has said otherwise.

- **E1, read or remove `initial_state_fields` and `prior_fields`: read (recommended default).**
  No rule condition reads a prior or initial state, but Lean's monitor does: it starts from the
  initial `StateValue` and matches transitions on the prior `StateValue`, fields included. Removing
  the fields would need a Lean redesign of `StateValue` in the correlated plan and its decode
  agreement proof (a Case such as the caller's start-to-close timeout has initial fields no result
  row reproduces), overlapping fn-93's Lean lanes. Reading them brings Go to Lean's semantics with no
  wire, Lean or fixture change. The cost is carrying fields in the Go monitor's state.
  Rejected: removal (above); a new PRIOR step field (a new concept).
- **D1, remove the dormant protocol arms: remove `evidence_field_id`, `correlated_capture` and
  `model_value`; keep the rest (recommended default).** The three removed arms have no producer, no
  fixture and no open spec; SCP-01 asks for capabilities that a concrete Temporal use case needs.
  The investigation showed `supporting_event_sequence` is set and read, `enumeration` is used by
  fn-89, the value-type arms and `Expression.any` are used by `ir` itself, and `elapsed_milliseconds`,
  `max_attempts` and `activity` are named by governed requirements (EVD-21, SEM-16, MOD-13), so none
  of those is dormant. The cost is one regeneration and one re-record.
- **D2, keep the facade wrapper structs (recommended default).** They are the public, read-only view
  a Driver programs against. Exporting `execution` types would widen the facade to save about 100
  lines.

The re-scoping also dropped three spec claims the investigation disproved: the D2 claim-validity
block is a re-check after a lock re-take, not a duplicate; `make umpire-gen-lean-api` does not
regenerate the Lean Testpilot protocol; and the Drivers' ceiling checks run at construction, not on
admitted data.

### Implementation tradeoffs

- **One opcode table** makes a new Opcode one row instead of edits to six switches. The price is
  indirection through function values in admission, which is not a hot path.
- **Validate once** trusts `Prepare`'s result inside Drivers. That is SEM-16's premise. The per-call
  authority checks stay because a Session call is not admitted data: it is a request that crosses
  the Driver boundary.
- **Shared fakes** couple tests across packages through one internal package. The coupling is kept
  small: one scripted Session with hooks, not a framework.

## Quick commands

```sh
go test -race -tags test_dep ./common/testing/testpilot/...
make umpire-check-case-runtime-conformance
make umpire-check-testpilot-protocol
make umpire-check-testpilot-authoring
make umpire-check-retired-vocabulary
make umpire-check-regression
make lint-code-fast
# lane G wire task only (live cluster):
make umpire-rerecord-pinned-runs
```





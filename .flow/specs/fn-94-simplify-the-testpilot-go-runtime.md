# fn-94-simplify-the-testpilot-go-runtime Simplify the Testpilot Go runtime

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

One lane changes the wire: lane G, only if the owner takes D1. Removing a oneof arm or field from the
hand-written `.proto` removes a Program or Contract capability. ART-04 needs no migration, because
the wire has no compatibility promise and no Case outside the repository exists (fn-87's decision,
restated by fn-89). SCP-01 argues for removal, since no concrete Temporal use case produces these
arms today.

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
   - 11 copies of `descriptorClosure`.
   - 12 copies of one `ProgramLimits` literal.
   - A ~190-line runtime fixture copied between `activation` and `worker`; only 40 lines differ.
   - 7 fake Sessions, 14 fake Drivers and 16 per-file Case builders.
   - Two scripted sessions in `tests/testcore/testpilot` that share most of their code.
   - Live-test run and assert boilerplate repeated three times.
5. **Dormant protocol surface.** Oneof arms that no Lean producer emits and no fixture uses, but
   that the Go runtime still handles. `Reference.evidence_field_id` has no producer at all, and
   `RunDiagnostic.supporting_event_sequence` is never read or set.

One finding may be a defect rather than bulk. `CorrelatedContract.initial_state_fields` and
`CorrelatedTransition.prior_fields` are emitted by Lean and present in 13 fixtures. Go admission
neither validates nor reads them: the STATE step condition reads `State` plus `StateFields` only.
Either they are dead wire fields, or a structured-state condition on the prior or initial state is
silently unsupported. EVD-04 (fail closed) makes an unread, unvalidated field an issue either way.

Done means the Testpilot Go code is smaller and there is less of it to read. Every behavior the
corpus, the conformance tests and the live identities pin stays the same.

## Architecture & Data Models
<!-- scope: technical -->

Lane E comes first. Lanes A, B, C, D1, D2 and F are mechanical and need no decision beyond this spec. Lanes G and H
carry owner decisions. Each lane is one or more tasks, each with its own receipt, which records the
measurement below before and after.

### Lane E: settle the unread correlated fields

- **E1.** Decide from the Lean producer (`Testpilot/Correlated.lean`, `Umpire/Case/Correlated.lean`)
  whether any rule can condition on a prior or initial structured state.
  - **If one can:** the Go correlated monitor reads `prior_fields` and `initial_state_fields`
    exactly as it reads `state_fields`. Admission validates them like `validModelValue`. A
    conformance entry is added through the Lean generator (ART-12), with a structured-state rule
    that is satisfied only when they are read.
  - **If none can:** the two fields leave the `.proto`, the Lean producer and the fixtures, in one
    commit (lane G's mechanics).
  - Either way, no admitted wire field goes unvalidated.

### Lane A: dead and test-only code

- **Unreachable:**
  - `worker.Driver.prepareDefinitionPlans`
  - `worker/routing.go`'s `validCoordinate`; `delivery` keeps its own
  - `PreparedProgram.PolicyIdentity`
  - `InstructionPlan.Assignments` and `AssignmentPlan`, `InstructionPlan.Input`
  - `PreparedContract.Snapshot`
  - `worker/carrier.go`'s `Handles` and `Quarantine`
  - the `prepared_case.go:41-46` type assertion to an anonymous `Evaluate` interface, whose error
    branch cannot happen. `MonitorFactory` gains `Evaluate`, or the prepared contract is stored
    beside it.
- **Test-only production code**, deleted and its tests re-pointed at the production path:
  - `worker/carrier.go` `ParentTerminal`; production uses `Session.parentTerminal`
  - `Session.preparedNexusDispatch`
  - `workerLease.release`
  - `Path.CheckFanout`
  - `Outage.Stopped`
  - `delivery.Activation.Handle`
  - `ProgramView.MaximumActivations` and the computation that feeds it
  - `assignment.environmentBindingID`
  - `PreparedContract.ProgramView`
  - `InstructionPlan.ResponseReads` and `ResponseReadPlan`
  - `Options.SessionOptions`/`Driver.Open`
  - the unprepared `newCompositeSession`
- **Facade surface with no caller in the repository:** `InstructionPlan.Guard`, `OutcomeType` and
  `Dependencies`, and the facade `Expression` type with `Expression.Evaluate`. The READMEs that
  document them are updated in the same commit, and so are the six aliases no non-test code outside
  the root package uses.
- **Untyped Nexus residue:**
  - `nexusResult.value`
  - the always-true `typed` map and its `future.Get(&Value)` fallback (`interpreter.go`,
    `typed.go:86`)
  - the `*testpilotspb.Value` input case (`sdk.go:103-104`) and `Value`-to-payload encoding
    (`callback.go:118-124`)
  - `PrepareNexus`'s `header`/`value` parameters, and `NexusDispatch.value`/`Value()`
  - two of the three reserved-header merges
- **Dead test helpers** that `deadcode -test` reports:
  - `runtimeText`, `runtimeStatusType`, `runtimeTextType` (both copies)
  - `openCountingDriver.Open`
  - `workflowAdmissionForTest`
  - the live-test trio `runCase`, `runCaseWithBinding` and `runBoundCase`, which `runCaptured*`
    duplicates and of which `runCase` has no caller

### Lane B: shared primitives

- **`ir` exports** `Invalid` (with the 256-byte path truncation), `ValidID`, `IsNil` and
  `CheckCeilings(limits, ceiling, path, optional...)`. `execution`, `verification`, the facade and
  the worker Driver call them in place of their copies.
  - `CheckCeilings` takes the error path as a parameter, so `execution`'s field-name path and
    `verification`'s `"contract"` path stay as they are.
- **One `temporal/internal` helper package** holds `nilValue`, `contextError`, the context-aware
  mutex, the effect-result clone, `nexusHeaderBytes`, the method-path builder, the StartWorkflow path
  constant and `hasWorkerEntrypoint`.
  - The mutex keeps the no-context `Lock()` that the server needs for completion.
- **Standard library replacements:** `slices.Contains`/`DeleteFunc` for `sessionIn`,
  `removeSessionFrom` and `slicesContains`; `slices.Collect(maps.Keys(...))` for `setKeys` and
  `nexusSetKeys`; `cmp.Or` for `firstError`.

### Lane C: validate once

- **Environment bindings.** Only `execution.bindPolicy` validates environment bindings. The facade's
  `BindingFingerprint` hashes the validated snapshot, and its bytes stay identical (ART-14).
  - `Prepare` clones the Profile once instead of calling `Snapshot().Snapshot()` and re-cloning it
    in `bindPolicy` and `bindRolePolicy`.
  - One sorted-unique-IDs helper serves both loops in `profile.go`.
  - `execution.Profile` stops hand-mirroring `ProfileSpec`.
- **Driver ceiling checks.** The worker's `validWorkerProfile` and the server's partial copy call
  the exported ceiling check. The worker's `validateSymbolicRoles` keeps only the worker-role-ID
  check and `profileRoleHasMethods`.
- **Carrier checks.** `delivery`'s `validateTopology`/`validateRoutes` and the worker carrier's
  "exactly one workflow reservation" check go: `execution/carrier.go` compiled the plan they
  re-check. `validateHandles` stays, because it checks runtime handles, and the plan-shape tests go
  with the checks.
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
  - Admission checks fire in today's order, because the corpus pins their paths.
- **Guarded input evaluation.** One `node.evaluateGuarded` replaces the three "guard, then input"
  evaluations (`request.go`, `scheduler.prepareInput`, `InstructionPlan.EvaluateInput`), with the
  same work accounting.
- **Scheduler and runtime helpers.**
  - One `takeCompletion` and one non-blocking drain replace the four copies of the completion-drain
    loop in the scheduler.
  - `execute` and `executeCleanup` stay separate, because EVD-14's stop semantics differ.
  - One close helper replaces the two cancel-and-close blocks in `runtime.go`.
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
  `authorizeUnary` prelude. The duplicated claim-validity block in the handle invoke path goes;
  this runs after fn-91, which renames that file's identifiers.
- **Workflow binding.** One comparable, JSON-tagged `delivery.WorkflowBinding` replaces the five
  representations of it. It serves as the route key and is what `PrepareRPC` returns, so the
  Marshal/Unmarshal round trips and reflection extraction in `driver.go` and `delivery/carrier.go`
  go. The JSON tags keep the route wire format byte-identical.
- **Admit paths (after fn-90 closes).**
  - `AdmitWorkflow` and `AdmitNexus` share one consume-and-admit tail.
  - `Driver.admitWorkflow` and `admitNexus` share one fan-out loop.
  - The Session's admission caches defer to the ledger's replay and conflict detection.
  - fn-90's live failures touch delivery, which is why this waits.

### Lane F: tests

- **Shared helpers.** One exported test helper for descriptor closures replaces the 11
  `descriptorClosure` copies, and one `ProgramLimits` fixture constructor replaces the 12
  literals.
- **Runtime fixture.** One runtime fixture, parameterized by reply-kind enum, command types and
  capture step, replaces the near-copies in `activation_test.go` and `runtime_fixture_test.go`.
- **Shared fakes.** An internal test support package holds one scripted fake Session, one fake Driver
  and fake effect and reservation types. Package-local fakes stay only where they test something
  specific:
  - `countingMonitorFactory`, the seam that `MonitorFactory` exists for;
  - the scheduler host;
  - the canary's fenced session.
- **`tests/testcore/testpilot`.** `workflowStartSession` and `artifactSession` become one scripted
  session with namespace, queue and history hooks. The unchanged-bytes checks share one helper.
- **Two-layer duplicates.** Where the facade and `execution` test the same environment rejections
  and concurrent preparations, the facade test stays. Exact duplicates go; focused tests that
  EVD-18 keeps independent of the corpus stay.
- **Live tests (after fn-90).** One helper covers the repeated-run assertion: distinct Run IDs, the
  workflow in its own namespace, Case bytes and frozen bindings unchanged. It serves the start,
  pair and caller live tests.
  - `nexusEvidenceKind` becomes a map, and `nexusOperationCoordinates` reads through a getter
    interface.
  - No live test is renamed, merged or removed, because fn-90 names these identities.

### Lane G (D1): dormant protocol surface

Remove the oneof arms that no Lean producer emits and no fixture uses, together with their Go
handlers, Lean `Testpilot.Authoring` builders and Lean `Testpilot.Correlated` cases:

- **`Reference.evidence_field_id`** has no producer. It is handled in
  `verification/correlated_prepare.go:349`, `correlated.go:336` and `Testpilot/Correlated.lean`.
- **`RunDiagnostic.supporting_event_sequence`** is never read or set.
- **Per-arm owner choice**, with removal recommended where no open spec names the arm:
  - `Reference.correlated_capture`, `Reference.model_value`
  - `Deadline.elapsed_milliseconds`
  - `InstructionLimits.max_attempts`
  - `Entrypoint.activity`
  - `Expression.any`
  - `ValueType.repeated`/`map`
  - `SingularType.enumeration`/`any`

  fn-79's deferred cancellation scope and fn-92's composition are checked for each arm before it
  goes.

Each removal regenerates the Go and Lean protocol code in the same commit (`make proto`, then
`make umpire-gen-lean-api`). It also updates the protocol extension checklist, and adds the retired
arm names to the retired-vocabulary gate (SEM-20). Case provenance stays: the Provenance
restatement makes those rows part of every Case, and "the runtime reads none of it" by design.

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
  - A lane G regeneration is the one allowed fixture change, and only for the removed arms.
- **Identity bytes.** `BindingFingerprint`, Driver identity (ART-14) and the delivery route wire
  format stay byte-identical, and a test pins each before its lane lands.
- **Concurrency.** Changes to the scheduler, delivery and ledger run with `-race`
  (`go test -race -tags test_dep`) and keep the EVD-14 and EVD-15 tests green. `execute` and
  `executeCleanup` stay apart, and so do the recorder's `publish`/`publishCleanup` and
  `admit`/`admitCleanup`. The same goes for `ir.inspect`'s two modes and `newEvaluator`'s
  view-equality guard.
- **Authority split.** No lane moves a check from the Driver's per-call authority boundary into
  admission, or back. Server and worker authority stay separate (EVD-17).
- **Concurrent specs.**
  - fn-91 renames the handle seam first. Lanes A, B, C and D1 touch no renamed identifier, and D2's
    server item waits for fn-91.
  - fn-90 owns delivery and the live tests until it closes, so D2's admit paths and F's live-test
    items wait for it.
  - fn-89 changes Contract Rule evaluation in `verification`, and lane D1's verification items
    rebase on it.
  - Lane G coordinates with fn-93's Lean lanes on `Testpilot/Authoring.lean`.
- **No new dependencies.** protovalidate is not in `go.mod`, and it would lose located paths and
  work accounting anyway. `deadcode` is used only as an offline investigation aid.
- **Explicitly not done.**
  - Table-driving the three small instruction-to-SDK switches, which would add code.
  - Typing role, slot or entrypoint IDs, which would ripple through `Coordinate` and the contract
    fn-91 edits.
  - Mapping context to outcome in one place: the three mappings use different codes.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** E1's decision is recorded in its task, and no field of an admitted Case is left
  unvalidated. Errors:
  - if the fields are read, a corpus entry fails when they are ignored;
  - if they are removed, admission rejects a Case that carries them.
- **R2:** None of the declarations lane A lists remains. `deadcode -test -tags 'test_dep integration'`
  over `./common/testing/testpilot/... ./tests/... ./tools/...` reports no unreachable function in
  Testpilot packages. Errors: a declaration it still reports is deleted or given a production
  caller, never suppressed.
- **R3:** No Testpilot package keeps a private copy of a primitive `ir` or the `temporal/internal`
  helper exports, and no hand-written `slices`/`maps`/`cmp` equivalent remains. Errors: an error path
  or ordering change fails R8.
- **R4:** Admitted data is validated once. The Drivers call the exported ceiling check, `delivery`
  does not re-check the compiled plan, and `Prepare` clones the Profile once, which a counting test
  pins. The Driver's per-call authority checks remain, each pinned by a rejection test.
- **R5:** One opcode table drives instruction binding and effect acceptance, and every static
  preparation rejection keeps its pinned `category` and `path`.
- **R6:** One `delivery.WorkflowBinding` exists, route wire bytes are unchanged, and the server and
  worker Drivers carry no Marshal/Unmarshal round trip for the binding.
- **R7:** Each shared test helper and fake has one definition, and every retained package-local fake
  says in one line what it tests that the shared one cannot. The live-test repeated-run helper serves
  the start, pair and caller tests, and the passing live identity count is unchanged.
- **R8:** `make umpire-check-case-runtime-conformance`, `make umpire-check-testpilot-protocol` and
  `make umpire-check-testpilot-authoring` pass with no fixture or `expected.json` change outside
  lane G. `go test -race -tags test_dep ./common/testing/testpilot/...` passes, and so do
  `make umpire-check-regression` (the current count of passing live identities) and
  `make lint-code-fast`.
- **R9:** For D1, the per-arm decision is recorded before removal. Each removed arm is gone from the
  `.proto`, the regenerated Go and Lean, the Lean builders and the Go handlers, and it is listed in
  the retired-vocabulary gate. The protocol extension checklist is updated. Errors: an arm that turns
  out to be emitted by a producer or named by an open spec blocks its removal.
- **R10:** The final receipt reports the measurement against the 2026-09-26 baseline:
  - handwritten Testpilot production at least 6% smaller (~1,100 lines);
  - its tests at least 5% smaller (~1,000 lines);
  - with lane G, the hand-written `.proto` at least 4% smaller.

  Errors: a floor missed is reported with the reason.

## Boundaries
<!-- scope: business -->

- No change to generated code (`api/testpilot/v1`, the generated Lean protocol) other than a lane G
  regeneration, and no change to the Case JSON fixtures other than lane G's.
- No Run, Verdict, Monitor or Contract semantics change, and no new Opcode or Profile field.
- No facade API change beyond removing the uncalled methods and aliases lane A lists.
- `tools/umpire` and `tools/canary` are out of scope, except where they call a removed declaration.
  A sweep of them is a separate spec.
- No live test renamed, merged or removed.
- No Lean change outside lane G and E1.

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

The Go code is tighter than the Lean model: this campaign is worth about 2,100 lines, against fn-93's
roughly 25,000. It is still worth running. Most of the value is in less code to read at the seam
every Driver implements, and in settling the unread wire fields.

### Owner decisions

- **E1, read or remove `initial_state_fields` and `prior_fields`: no default.** The Lean producer
  decides. E1's first step is to find out whether any rule can condition on a prior or initial
  structured state.
- **D1, remove the dormant protocol arms: recommended for `evidence_field_id` and
  `supporting_event_sequence`, per arm for the rest.** No producer emits them and no fixture uses
  them, and SCP-01 asks for capabilities that a concrete Temporal use case needs. The cost is a
  protocol regeneration per batch. The rest are small language features (`any`, `repeated`, `map`,
  `activity`) that fn-79 or a later feature might want. Removing them now and restoring one on
  demand is cheaper than carrying all of them, but the owner may prefer to keep the value-type arms.
- **D2, keep the facade wrapper structs: recommended.** They are the public, read-only view a Driver
  programs against. Exporting `execution` types would widen the facade to save about 100 lines.

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
make umpire-check-regression
make lint-code-fast
```

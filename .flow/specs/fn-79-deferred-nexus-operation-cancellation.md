# Nexus operation cancellation — deferred

User decision: defer Nexus operation cancellation until explicitly requested again. This is outside fn-78, fn-77, and fn-70 completion requirements. The autonomous delivery goal is not approval to resume it.

## Scope

Preserve correlated cancellation evidence, per-operation SDK cancellation capability, and authored Nexus cancellation qualification. Re-plan against completed generic scoped monitoring before resuming. Existing Run/context shutdown, bounded cleanup, and regression protection are not deferred.

## Requirements

- **R1:** Correlated cancellation adapter distinguishes submission from confirmation, retains exact scope/source/Run Event support, and accepts canceled or completed without forcing outcomes.
- **R2:** One authorized per-operation cancellation capability remains separate from Workflow cancellation and activation shutdown, with static admission and race tests.
- **R3:** Authored cancellation produces a deterministic admitted Case and qualifies both resolutions through public Prepare/Run; invalid, incomplete, concurrent, and cleanup-failure evidence preserves correct verdicts.
- **R4:** Parameterized cancellation identity/confirmation qualification formerly illustrated in fn-77 is deferred with this work. Reuse the generic typed-operation and field semantics once available.

## Preserved detailed requirements

The following original task descriptions are retained as deferred requirements, not active work or completion claims. Generic syntax, diagnostics, documentation, and non-cancellation qualification formerly bundled in task .8 remain in fn-78.

### Former fn-78.5

---
satisfies: [R3, R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.5 Project correlated Nexus evidence into semantic steps

## Description
Adapt the generic D3 projection kernel to correlated Temporal Nexus history. The adapter establishes cancellation confirmation and either terminal resolution from declared causal evidence while leaving SDK transport in `common/testing/testpilot/temporal/worker` and all generic Testpilot packages free of Nexus semantics.

**Size:** M
**Files:** `model/Temporal/System/Nexus/{Core,ImplementationLink,ImplementationLinkTests}.lean`, focused Nexus evidence modules/tests under `model/Temporal/System/Nexus/`, `model/TemporalModelTests/Nexus/ImplementationLink.lean`, `model/Temporal/Feature/Nexus/Success/{Model,Tests}.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/TemporalModelTests/Nexus/ImplementationLink.lean, model/Temporal/Feature/Nexus/Success/Model.lean, model/Temporal/Feature/Nexus/Success/Tests.lean]

### Approach
- Declare correlation over namespace, workflow/run, scheduled-event/operation, and request identity using stable source-event identity and causal references.
- Model cancellation submission as authority for the SDK effect only. Emit cancellation-requested semantics only from correlated confirmation evidence.
- Preserve canceled and completed as alternative Target-owned resolutions and keep operation cancellation separate from workflow cancellation and activation shutdown.
- Project only the closed evidence fields/support required by the checked declaration; keep raw history and callback mechanics in the Temporal worker adapter delivered by task 1.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/ImplementationLink.lean` — sole System/Feature correspondence leaf
- `model/Temporal/System/Nexus/Core.lean` — checked System lifecycle
- `model/Temporal/Feature/Nexus/Success/Model.lean` — feature Target authority
- `model/Temporal/Feature/Nexus/Success/Producer.lean:246-300` — existing checked success producer gate
- `common/testing/testpilot/temporal/worker/callback.go` — SDK-only Nexus mechanics boundary

### Key context
- Temporal cancellation is advisory; a handler may ignore it and complete.
- The server-package no-Nexus dependency test from task 1 remains a hard boundary.
## Acceptance
- [ ] Submitting cancellation alone emits no cancellation-confirmed semantic step; only correlated confirmation evidence can emit it.
- [ ] Canceled and completed remain alternative model-owned terminal resolutions, and the adapter never forces the exploration-selected outcome.
- [ ] Correlation uses declared namespace, workflow/run, scheduled-event/operation, request, stable source-event, and causal identities with no wall-clock ordering.
- [ ] Duplicate/irrelevant evidence stutters, missing parents remain pending, and wrong-operation, conflicting, unsupported, cyclic, or invalid-step evidence rejects atomically with exact support retained for earlier emissions.
- [ ] Per-operation cancellation handles remain distinct from workflow cancellation and activation shutdown.
- [ ] Focused System/Feature correspondence tests cover both resolutions, incomplete and adversarial evidence, concurrent Run isolation, and immutable prior violations.
- [ ] `common/testing/testpilot/temporal/server` remains free of Nexus identifiers and dependencies; final ownership documentation is updated by the qualification task.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Former fn-78.8

---
satisfies: [R6, R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.8 Qualify authored Nexus cancellation through Testpilot

## Description
Expose D5's readable bounded temporal syntax over the checked scoped clause and qualify a complete authored Nexus cancellation Case through the existing public Testpilot Prepare/Run facade. Keep Nexus syntax and lowering next to the feature adapter while reusable temporal constructors remain in `Umpire.Property`.

**Size:** L
**Files:** `model/Umpire/Property/{Authoring,Syntax,Tests/**}.lean`, `model/Temporal/Feature/Nexus/Success/{Authoring,Syntax,Model,Producer,Tests,Integration.md}`, `model/Temporal/Tool/Testpilot.lean`, managed Testpilot fixtures/generator tests, `tests/testpilot_async_nexus_case_test.go`, `model/{README,ARCHITECTURE}.md`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/Umpire/Property/**, model/Temporal/Feature/Nexus/Success/**, model/Temporal/Tool/Testpilot.lean, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/**, tests/testpilot_async_nexus_case_test.go, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Add typed constructors and hygienic notation that resolve trigger, response, same-operation key, operation-transition clock, natural bound, and endpoint explicitly into the same checked clause.
- Use elaboration for context-sensitive rejection and source-local diagnostics; do not add a general expression framework or make authors maintain serialization/proof plumbing.
- Extend the existing Nexus success Producer's checked Query/witness gate to cancellation, compile one deterministic admitted Case, and bind task 9's per-operation cancellation capability through task 1's generic server seam.
- Drive local Temporal integration through public `Prepare`/`Run`; recognize correlated cancellation and either permitted terminal result without forcing the chosen model outcome.
- Update ownership/architecture docs and replace the Nexus success integration draft's unsupported-cancellation statements.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Authoring.lean` — reusable typed Property owner
- `model/Temporal/Feature/Nexus/Success/Syntax.lean` — feature-local macro pattern
- `model/Temporal/Feature/Nexus/Success/Producer.lean:246-300` — checked Query/witness and Case compilation
- `model/Temporal/Feature/Nexus/Success/Tests.lean` — authored equality/rejection tests
- `tests/testpilot_async_nexus_case_test.go` — existing real Driver proof

### Key context
- Lean 4 elaborators should issue context/type/key/clock errors at the author syntax; macros should handle syntax-only expansion.
- Parameterized field expressions remain fn-77-owned; consume delivered interfaces without duplicating them and keep this label-only qualification independently complete.
## Acceptance
- [ ] Readable bounded temporal notation exposes or unambiguously resolves trigger, response, correlation key, semantic clock, natural bound, and endpoint policy and elaborates to the same canonical checked clause as typed constructors.
- [ ] Compile-failure tests reject state/step/trace/evidence context misuse, raw evidence or command effects in Properties, wrong keys/clocks/scopes/references, ambiguous names, incompatible bounds, and unsupported formulas at the author expression.
- [ ] Ordinary authors supply semantic choices only; derived IDs, capability version, provenance, and generated monitor mechanics remain inspectable without handwritten duplicate monitor rules.
- [ ] A checked Nexus cancellation Query produces deterministic admitted Case 1.0 bytes and runs through the existing public Prepare/Run facade with the authorized generic capability.
- [ ] Local integration recognizes correlated cancellation confirmation followed by canceled or completed without forcing the selected model result; submission alone cannot satisfy the Property.
- [ ] Controlled violating, incomplete, wrong-correlation, stopped/lost execution, and cleanup-failure fixtures preserve the distinction among violation, inconclusive, and disposition while retaining earlier proof.
- [ ] Repeated/concurrent Runs and tenfold candidate/evidence/overlapping-obligation loads demonstrate isolation and bounded failure with recorded semantic/work limits and costs.
- [ ] Existing success/rejection/cross-language/identity/lifecycle regressions remain green; unchanged fixtures retain exact bytes and IDs.
- [ ] `make umpire-build-model`, `make umpire-check-regression`, `make lint-model`, `make lint-code`, generated staleness checks, focused Go tests with `-tags test_dep`, and the scoped integration test with `test_dep integration` pass with evidence recorded.
- [ ] Umpire and model architecture docs describe the final ownership and syntax; `model/Temporal/Feature/Nexus/Success/Integration.md` no longer claims cancellation lowering is unsupported.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Former fn-78.9

---
satisfies: [R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.9 Add per-operation cancellation capability

## Description
Add the minimal generic per-operation cancellation instruction and capability required by D4/D5. Testpilot authorizes cancellation of one previously started operation; the Temporal worker owns the SDK cancellation handle and submits cancellation without treating submission as semantic confirmation or forcing the final result.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{instruction,program}.proto`, generated Testpilot Go/Lean protocol files, `model/Testpilot/Authoring.lean`, `common/testing/testpilot/internal/execution/{contracts,program,prepare,scheduler}*.go`, `common/testing/testpilot/temporal/worker/{interpreter,session,driver,runtime_fixture_test,runtime_test,sdk_test}.go`
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/instruction.proto, proto/internal/temporal/server/api/testpilot/v1/program.proto, api/testpilot/v1/**, model/Testpilot/Authoring.lean, common/testing/testpilot/internal/execution/**, common/testing/testpilot/temporal/worker/**]

### Approach
- Add one feature-neutral instruction referencing an admitted in-flight operation/effect identity; do not add a Nexus branch to generic server transport.
- Admit the reference, dependency, entrypoint context, capability claim, and work bounds statically before Driver I/O.
- Have the Temporal worker create and retain a distinct SDK cancel function with each started Nexus operation, invoke exactly that handle, and keep workflow cancellation and activation shutdown separate.
- Return only effect-submission status. Cancellation confirmation and canceled/completed resolution remain observation-owned semantic evidence.
- Make completion-versus-cancellation, duplicate cancellation, close, and cleanup races deterministic and bounded; never retry automatically.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` — current closed instruction union
- `common/testing/testpilot/internal/execution/contracts.go` — admitted opcode/capability contracts
- `common/testing/testpilot/internal/execution/scheduler.go` — effect scheduling and cancellation ownership
- `common/testing/testpilot/temporal/worker/interpreter.go` — SDK instruction execution
- `common/testing/testpilot/temporal/worker/sdk_test.go:364-389` — current async Nexus future path

### Key context
- The instruction authorizes cancellation submission only; a correlated history event is required for the model step.
- Per-operation handles must survive only for their owning Run/activation and must not become workflow or server-global state.

## Acceptance
- [ ] A closed feature-neutral instruction can cancel one admitted in-flight operation/effect by stable identity through the existing generic server capability seam.
- [ ] Prepare rejects missing, duplicate, cross-entrypoint, wrong-kind, unbounded, or unauthorized cancellation references before Driver I/O.
- [ ] The Temporal worker stores a distinct SDK cancel function for each started Nexus operation and never substitutes workflow cancellation or activation shutdown.
- [ ] Cancellation submission produces effect status only and cannot emit cancellation-confirmed or terminal semantic steps.
- [ ] Tests cover cancel-before-await, completion-before-cancel, cancellation/completion races, duplicate/late cancel, missing handle, Run close, cleanup, and concurrent operation isolation without deadlock or automatic retry.
- [ ] Existing completion and success behavior remains valid; server dependency tests stay Nexus-free.
- [ ] Protocol/Lean generation, focused admission/worker tests with `-tags test_dep`, compatibility fixtures, and scoped lints pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

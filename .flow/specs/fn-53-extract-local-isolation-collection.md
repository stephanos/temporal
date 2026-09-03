# Extract local isolation collection state machine

## Overview

Separate the invocation-local isolation collection and pre-probe decision logic from Temporal environment orchestration. The refactor creates one private, concrete module whose small interface hides ordered count collection, closure, invalidation, and the one-shot isolation decision while preserving every existing observable behavior.

## Goal & Context

The local Temporal environment currently combines lifecycle orchestration, SDK probing, receipt construction, synchronization, and a closed in-memory collection protocol. Extracting the protocol improves locality and makes its fail-closed transitions directly testable without changing behavior for Umpire users or operators.

Affected stakeholders:
- Developers gain a focused module and deterministic unit-test surface.
- End users see no behavior, command, receipt, or diagnostic changes.
- Operations gain no configuration, deployment, persistence, or monitoring surface.

## Scope

- Move operation/control count recording, collection closure, invalidation, and the one-shot pre-probe decision behind one private concrete module.
- Initialize that module once with the exact expected commands and operation correlation.
- Keep lifecycle entry points, the existing mutex, context handling, isolation-command validation, SDK probing, and receipt construction in the Temporal environment.
- Add focused transition tests plus environment-level precedence and receipt regressions while retaining the existing lifecycle and attached-authority coverage.

## Architecture & Data Models

The extracted module owns only run-local collection state and immutable expected identities. Its interface accepts the three existing collection operations and produces one closed pre-probe outcome: failed, canceled, or ready for the existing execution probe. It is a concrete internal module, not a new Go interface or adapter seam.

The Temporal environment remains the synchronization and orchestration owner. Each existing collection entry point holds the environment mutex while delegating to the module. Isolation continues to validate context and the isolation command before consulting the module; only a ready outcome permits the existing SDK probe, after which the environment constructs the existing receipt.

## API Contracts

- Existing environment method shapes and their callers remain unchanged.
- The module is unexported and construction requires the expected prepare, realize, and observe commands plus the operation correlation; no useful zero-value contract is introduced.
- Record and close failures retain their exact diagnostics and permanently invalidate the collection.
- The pre-probe decision is one-shot and closed over three outcomes. It does not receive a context, call Temporal, construct receipts, or expose its internal fields.

## Edge Cases & Constraints

- Wrong commands, wrong correlations, duplicate records, duplicate close, and mutations after close preserve their current mutation-before-error and permanent-invalidation behavior.
- Invalidation and counts greater than one dominate every cancellation condition, including open or otherwise incomplete collections. A repeated isolation decision invalidates the collection and fails. Only when neither failure condition applies do missing records, zero counts, or an open collection cancel; exactly one operation and one control count in a closed valid collection is ready for probing.
- Nil or canceled context and an unsupported isolation command retain their current precedence outside the module. A missing probe cancels; a probe error keeps its current context-aware failure mapping.
- The existing environment mutex remains the sole synchronization seam. State transitions stay bounded O(1), add no I/O or blocking, and remain deterministic under increased call volume.
- State remains invocation-local and non-durable. A process crash gains no recovery or checkpoint semantics.
- Exact opaque command and correlation comparisons remain authority checks; the refactor must not broaden accepted identities.
- Existing comments are preserved unless their ownership statement would become false after the move.

## Approach

Implement one M-sized extraction: place the private collection module and its focused tests beside the environment, delegate the existing methods through it, and leave probe and receipt behavior in the environment. Use table-driven tests at the module interface for all transition and decision classes, add environment-level tests for orchestration precedence and exact receipts, then run the existing package tests as the regression check.

## Quick commands

```bash
go test -tags test_dep ./tools/umpire/temporal/local
make lint-code
```

## Acceptance Criteria

- **R1:** Isolation collection state and pre-probe decisions are owned by one unexported concrete module initialized with the exact expected commands and operation correlation; existing environment method shapes remain unchanged and no exported type, Go interface, configuration, dependency, or durable state is added. Errors: incomplete internal initialization cannot be accepted as a valid collection.
- **R2:** Record and close transitions retain exact command/correlation checks, error strings, mutation-before-error ordering, duplicate handling, post-close rejection, and permanent invalidation. Errors: wrong command, wrong correlation, duplicate operation record, duplicate control record, duplicate close, and every mutation after close are covered explicitly.
- **R3:** The one-shot pre-probe decision preserves the exact failed/canceled/ready partition and precedence: prior invalidation, repeated decision, and counts above one fail before any cancellation condition is considered; only in the absence of a failure do missing records, zero counts, and missing closure cancel; one recorded operation and one recorded control in a closed valid collection is ready. Errors: invalid-plus-incomplete, count-above-one-plus-open, and count-above-one-plus-missing-input combinations all fail rather than cancel.
- **R4:** Context handling, isolation-command validation, probe execution, and receipt construction remain in the environment with unchanged precedence, statuses, codes, facts, correlations, and diagnostics. Errors: nil or canceled context, unsupported isolation command, missing probe, probe failure, and cross-run or crossed-correlation inputs preserve current outcomes.
- **R5:** Focused table-driven tests exercise the module interface, and environment-level regressions prove every R4 precedence path with probe-call counts, decision-consumption behavior, and exact receipt status, code, facts, and correlations; the existing local lifecycle and attached-authority tests continue to pass, and live ownership documentation remains accurate without an edit unless implementation ownership changes. Errors: any uncovered R4 path, race-prone synchronization change, or documentation drift fails verification rather than being accepted as cleanup.

## Early proof point

The sole implementation task validates the seam by passing the pure transition table and the unchanged environment lifecycle tests. If either exposes a semantic or diagnostic drift, keep the logic in the environment and reconsider the module's decision interface before proceeding.

## Boundaries

- No decomposition of the Lean Observation evaluator, artifact validators, or the broader local Temporal authority implementation.
- No changes to portable-plan authority, caller-selected adapters, or work owned by the caller-neutral gRPC portable-test-plan spec.
- No new runtime abstraction, exported API, compatibility facade, configuration, persistence, metrics, or CI workflow.
- No normative Umpire rule or generated-file changes.

## Decision Context

- Chose the isolation collection seam because it concentrates a closed state protocol behind a small interface and has no file overlap with open implementation specs.
- Kept synchronization in the environment so state decisions remain serialized with probe execution and receipt construction; an independently synchronized module would add ordering risk.
- Rejected a generic runtime isolation interface as hypothetical indirection: only the local Temporal authority owns this protocol.
- Rejected a broader environment split as navigation-only churn, and deferred the larger Observation evaluator decomposition to a separate future spec.
- The extra internal file adds minor structural complexity in exchange for higher test leverage and locality. Runtime cost remains constant, scalability is unchanged, and exact identity checks preserve the existing authority boundary.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Private initialized collection module with stable caller surface | Extract and test local isolation collection state machine | — |
| R2 | Exact record/close transition and diagnostic compatibility | Extract and test local isolation collection state machine | — |
| R3 | Exact one-shot failed/canceled/ready decision | Extract and test local isolation collection state machine | — |
| R4 | Unchanged orchestration, probing, and receipts | Extract and test local isolation collection state machine | — |
| R5 | Focused and regression verification with documentation check | Extract and test local isolation collection state machine | — |

## References

- Umpire 4 rules MOD-06, MOD-07, and MOD-08 require small cohesive interfaces, explicit responsibilities, and isolated testability.
- Umpire 4 rules EVD-04 through EVD-06 require fail-closed Evidence handling, independent stage statuses, and linked execution receipts.
- Completed cleanup specs fn-44 and fn-49 establish related fail-closed ordering and diagnostic-preservation precedent without sharing this implementation surface.
